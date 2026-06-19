package handler

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/webdad/media-service/internal/config"
	"github.com/webdad/media-service/internal/imageproc"
	"github.com/webdad/media-service/internal/logging"
	"github.com/webdad/media-service/internal/middleware"
	"github.com/webdad/media-service/internal/storage"
	"github.com/webdad/media-service/internal/validate"
)

// roleAdmin : valeur du claim `role` autorisant la suppression de n'importe
// quel média ET le bypass des caps de taille à l'upload (aligné sur
// user-service / profil-service).
const roleAdmin = "admin"

// sniffLen : nombre d'octets lus en tête pour la détection MIME réelle.
// mimetype recommande ≥ 3072 octets.
const sniffLen = 3072

// maxTransformBytes borne le traitement image en mémoire. Les utilisateurs
// ordinaires sont déjà plafonnés à 5 Mo ; cette limite protège surtout le
// bypass admin contre les très gros originaux.
const maxTransformBytes = 20 * 1024 * 1024

// MediaHandler sert les endpoints média au-dessus du stockage objet.
type MediaHandler struct {
	store         *storage.Store
	maxImageBytes int64
	maxVideoBytes int64
	maxBlobBytes  int64
}

// NewMediaHandler construit le handler avec ses caps de taille (appliqués aux
// utilisateurs non-admin ; les admins bypassent, cf. Upload/UploadEncrypted).
func NewMediaHandler(store *storage.Store, cfg *config.Config) *MediaHandler {
	return &MediaHandler{
		store:         store,
		maxImageBytes: cfg.MaxImageBytes,
		maxVideoBytes: cfg.MaxVideoBytes,
		maxBlobBytes:  cfg.MaxBlobBytes,
	}
}

// uploadResponse : corps renvoyé après un upload réussi. `url` est RELATIVE à
// la gateway (`/media/<id>`) ; le front la préfixe de l'URL de la gateway.
type uploadResponse struct {
	ID       string                    `json:"id"`
	URL      string                    `json:"url"`
	Mime     string                    `json:"mime"`
	Kind     string                    `json:"kind"`
	Size     int64                     `json:"size"`
	Width    int                       `json:"width,omitempty"`
	Height   int                       `json:"height,omitempty"`
	Variants map[string]variantPayload `json:"variants,omitempty"`
}

type variantPayload struct {
	URL    string `json:"url"`
	Mime   string `json:"mime"`
	Size   int64  `json:"size"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// Upload : POST /media (multipart, champ `file`). Valide le type réel (magic
// bytes) + la taille, range dans MinIO sous un id aléatoire, renvoie 201.
// @Summary     Uploader un média (image ou vidéo)
// @Tags        media
// @Accept      multipart/form-data
// @Produce     json
// @Security    BearerAuth
// @Param       file formData file true "Fichier image (JPEG/PNG/GIF/WebP) ou vidéo (MP4/WebM)"
// @Success     201 {object} handler.uploadResponse
// @Failure     400 {object} map[string]string
// @Failure     401 {object} map[string]string
// @Failure     413 {object} map[string]string "Fichier trop volumineux"
// @Failure     415 {object} map[string]string "Type MIME non supporté"
// @Router      /media [post]
func (h *MediaHandler) Upload(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "non authentifié"})
		return
	}

	// Les administrateurs ne sont pas plafonnés : ils peuvent uploader des
	// médias plus lourds (bypass des caps de taille).
	isAdmin := claims.Role == roleAdmin

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "champ `file` manquant"})
		return
	}

	// Rejet précoce si la taille dépasse même le plus grand cap (vidéo), avant
	// de lire le moindre octet. Ignoré pour les admins (non plafonnés).
	if !isAdmin && fileHeader.Size > h.maxVideoBytes {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": validate.ErrTooLarge.Error()})
		return
	}

	f, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "fichier illisible"})
		return
	}
	defer func() { _ = f.Close() }()

	// Lecture de la tête pour la détection MIME, puis reconstruction du flux
	// complet (tête + reste) pour le streaming vers MinIO sans tout charger en
	// mémoire.
	head := make([]byte, sniffLen)
	n, err := io.ReadFull(f, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lecture du fichier impossible"})
		return
	}
	head = head[:n]

	mime, kind, err := validate.Detect(head)
	if err != nil {
		logging.FromGin(c).Warn("upload refusé : type MIME non supporté", "error", err)
		c.JSON(http.StatusUnsupportedMediaType, gin.H{"error": err.Error()})
		return
	}
	if !isAdmin {
		if err := validate.CheckSize(kind, fileHeader.Size, h.maxImageBytes, h.maxVideoBytes); err != nil {
			logging.FromGin(c).Warn("upload refusé : fichier trop volumineux", "kind", string(kind), "size", fileHeader.Size)
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": err.Error()})
			return
		}
	}

	id, err := randomID()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "génération d'identifiant impossible"})
		return
	}

	data, err := io.ReadAll(io.MultiReader(bytes.NewReader(head), f))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lecture du fichier impossible"})
		return
	}

	if err := h.store.Put(c.Request.Context(), id, bytes.NewReader(data), int64(len(data)), mime, claims.UserID); err != nil {
		logging.FromGin(c).Error("échec stockage MinIO (upload)", "error", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "échec du stockage"})
		return
	}

	width, height := 0, 0
	variants := map[string]variantPayload{}
	if kind == validate.KindImage && int64(len(data)) <= maxTransformBytes {
		info, generated, err := imageproc.GenerateVariants(data, mime)
		if err != nil {
			logging.FromGin(c).Warn("variantes image non générées", "media_id", id, "mime", mime, "error", err)
		} else {
			width, height = info.Width, info.Height
			for _, v := range generated {
				if err := h.store.PutVariant(c.Request.Context(), id, v.Name, bytes.NewReader(v.Bytes), int64(len(v.Bytes)), v.Mime, claims.UserID); err != nil {
					logging.FromGin(c).Warn("variante image non stockée", "media_id", id, "variant", v.Name, "error", err)
					continue
				}
				variants[v.Name] = variantPayload{
					URL:    "/media/" + id + "/" + v.Name,
					Mime:   v.Mime,
					Size:   int64(len(v.Bytes)),
					Width:  v.Width,
					Height: v.Height,
				}
			}
		}
	}
	if len(variants) == 0 {
		variants = nil
	}

	logging.FromGin(c).Info("média uploadé", "media_id", id, "mime", mime, "size", fileHeader.Size)
	c.JSON(http.StatusCreated, gin.H{"data": uploadResponse{
		ID:       id,
		URL:      "/media/" + id,
		Mime:     mime,
		Kind:     string(kind),
		Size:     int64(len(data)),
		Width:    width,
		Height:   height,
		Variants: variants,
	}})
}

// UploadEncrypted : POST /media/encrypted (multipart, champ `file`). Destiné
// aux pièces jointes de messagerie E2EE : le corps est un BLOB CHIFFRÉ côté
// client (octets aléatoires) → aucune détection MIME possible ni souhaitable.
// On valide donc uniquement la taille (cap vidéo, le plus large) et on stocke
// en `application/octet-stream`. La vraie nature (image/vidéo, nom, nonce) vit
// dans l'enveloppe chiffrée du message, jamais ici → serveur aveugle.
// @Summary     Uploader un blob chiffré E2EE
// @Tags        media
// @Accept      multipart/form-data
// @Produce     json
// @Security    BearerAuth
// @Param       file formData file true "Blob chiffré (contenu opaque)"
// @Success     201 {object} handler.uploadResponse
// @Failure     400 {object} map[string]string
// @Failure     401 {object} map[string]string
// @Router      /media/encrypted [post]
func (h *MediaHandler) UploadEncrypted(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "non authentifié"})
		return
	}

	// Admins non plafonnés (bypass du cap blob), comme pour les médias en clair.
	isAdmin := claims.Role == roleAdmin

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "champ `file` manquant"})
		return
	}
	if !isAdmin && fileHeader.Size > h.maxBlobBytes {
		logging.FromGin(c).Warn("upload chiffré refusé : blob trop volumineux", "size", fileHeader.Size)
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": validate.ErrTooLarge.Error()})
		return
	}

	f, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "fichier illisible"})
		return
	}
	defer func() { _ = f.Close() }()

	id, err := randomID()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "génération d'identifiant impossible"})
		return
	}

	if err := h.store.Put(c.Request.Context(), id, f, fileHeader.Size, "application/octet-stream", claims.UserID); err != nil {
		logging.FromGin(c).Error("échec stockage MinIO (upload chiffré)", "error", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "échec du stockage"})
		return
	}

	logging.FromGin(c).Info("blob chiffré E2EE uploadé", "media_id", id, "size", fileHeader.Size)
	c.JSON(http.StatusCreated, gin.H{"data": uploadResponse{
		ID:   id,
		URL:  "/media/" + id,
		Mime: "application/octet-stream",
		Kind: "blob",
		Size: fileHeader.Size,
	}})
}

// Download : GET /media/:id — public (id non devinable). Stream depuis MinIO
// via http.ServeContent (gère Range/seek vidéo, HEAD, If-None-Match). Cache
// long + immutable : un id correspond toujours au même contenu.
// @Summary     Télécharger un média (lecture publique, streaming + Range)
// @Tags        media
// @Produce     application/octet-stream
// @Param       id path string true "Media ID"
// @Success     200 {file} string "Contenu binaire"
// @Success     206 {file} string "Contenu partiel (Range)"
// @Failure     404 {object} map[string]string
// @Router      /media/{id} [get]
func (h *MediaHandler) Download(c *gin.Context) {
	id := c.Param("id")
	h.serveObject(c, id)
}

// DownloadVariant : GET /media/:id/:variant — sert une variante générée.
func (h *MediaHandler) DownloadVariant(c *gin.Context) {
	id := c.Param("id")
	variant := c.Param("variant")
	if !validVariant(variant) {
		c.JSON(http.StatusNotFound, gin.H{"error": "média introuvable"})
		return
	}
	h.serveObject(c, storage.VariantKey(id, variant))
}

func (h *MediaHandler) serveObject(c *gin.Context, id string) {
	obj, info, err := h.store.Open(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "média introuvable"})
			return
		}
		logging.FromGin(c).Error("erreur lecture MinIO (download)", "media_id", id, "error", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "lecture impossible"})
		return
	}
	defer func() { _ = obj.Close() }()

	header := c.Writer.Header()
	if info.ContentType != "" {
		header.Set("Content-Type", info.ContentType)
	}
	header.Set("Cache-Control", "public, max-age=31536000, immutable")
	if info.ETag != "" {
		header.Set("ETag", `"`+info.ETag+`"`)
	}

	http.ServeContent(c.Writer, c.Request, id, info.LastModified, obj)
}

// Delete : DELETE /media/:id — propriétaire ou admin uniquement.
// @Summary     Supprimer un média (propriétaire ou admin)
// @Tags        media
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "Media ID"
// @Success     204
// @Failure     401 {object} map[string]string
// @Failure     403 {object} map[string]string
// @Failure     404 {object} map[string]string
// @Router      /media/{id} [delete]
func (h *MediaHandler) Delete(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "non authentifié"})
		return
	}
	id := c.Param("id")

	info, err := h.store.Stat(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "média introuvable"})
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": "lecture impossible"})
		return
	}

	if storage.OwnerOf(info) != claims.UserID && claims.Role != roleAdmin {
		logging.FromGin(c).Warn("suppression média refusée : non propriétaire", "media_id", id)
		c.JSON(http.StatusForbidden, gin.H{"error": "média non détenu"})
		return
	}

	if err := h.store.RemoveMediaSet(c.Request.Context(), id); err != nil {
		logging.FromGin(c).Error("erreur suppression MinIO", "media_id", id, "error", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "suppression impossible"})
		return
	}
	c.Status(http.StatusNoContent)
}

func validVariant(name string) bool {
	switch name {
	case "thumb", "small", "medium", "large":
		return true
	default:
		return false
	}
}

// PurgeByOwner : DELETE /media/owners/:id — efface TOUS les objets d'un
// propriétaire (effacement RGPD, admin via middleware). Renvoie le nombre
// d'objets supprimés.
func (h *MediaHandler) PurgeByOwner(c *gin.Context) {
	n, err := h.store.RemoveByOwner(c.Request.Context(), c.Param("id"))
	if err != nil {
		logging.FromGin(c).Error("erreur purge MinIO (RGPD)", "target_id", c.Param("id"), "error", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "purge impossible"})
		return
	}
	logging.FromGin(c).Info("médias purgés RGPD (admin)", "target_id", c.Param("id"), "objects_deleted", n)
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"objects_deleted": n}})
}

// randomID génère un identifiant opaque non devinable (128 bits → 32 hex).
func randomID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
