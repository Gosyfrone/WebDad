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
	"github.com/webdad/media-service/internal/middleware"
	"github.com/webdad/media-service/internal/storage"
	"github.com/webdad/media-service/internal/validate"
)

// roleAdmin : valeur du claim `role` autorisant la suppression de n'importe
// quel média (aligné sur user-service / profil-service).
const roleAdmin = "admin"

// sniffLen : nombre d'octets lus en tête pour la détection MIME réelle.
// mimetype recommande ≥ 3072 octets.
const sniffLen = 3072

// MediaHandler sert les endpoints média au-dessus du stockage objet.
type MediaHandler struct {
	store         *storage.Store
	maxImageBytes int64
	maxVideoBytes int64
}

// NewMediaHandler construit le handler avec ses caps de taille.
func NewMediaHandler(store *storage.Store, cfg *config.Config) *MediaHandler {
	return &MediaHandler{
		store:         store,
		maxImageBytes: cfg.MaxImageBytes,
		maxVideoBytes: cfg.MaxVideoBytes,
	}
}

// uploadResponse : corps renvoyé après un upload réussi. `url` est RELATIVE à
// la gateway (`/media/<id>`) ; le front la préfixe de l'URL de la gateway.
type uploadResponse struct {
	ID   string `json:"id"`
	URL  string `json:"url"`
	Mime string `json:"mime"`
	Kind string `json:"kind"`
	Size int64  `json:"size"`
}

// Upload : POST /media (multipart, champ `file`). Valide le type réel (magic
// bytes) + la taille, range dans MinIO sous un id aléatoire, renvoie 201.
func (h *MediaHandler) Upload(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "non authentifié"})
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "champ `file` manquant"})
		return
	}

	// Rejet précoce si la taille dépasse même le plus grand cap (vidéo), avant
	// de lire le moindre octet.
	if fileHeader.Size > h.maxVideoBytes {
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
		c.JSON(http.StatusUnsupportedMediaType, gin.H{"error": err.Error()})
		return
	}
	if err := validate.CheckSize(kind, fileHeader.Size, h.maxImageBytes, h.maxVideoBytes); err != nil {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": err.Error()})
		return
	}

	id, err := randomID()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "génération d'identifiant impossible"})
		return
	}

	reader := io.MultiReader(bytes.NewReader(head), f)
	if err := h.store.Put(c.Request.Context(), id, reader, fileHeader.Size, mime, claims.UserID); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "échec du stockage"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": uploadResponse{
		ID:   id,
		URL:  "/media/" + id,
		Mime: mime,
		Kind: string(kind),
		Size: fileHeader.Size,
	}})
}

// UploadEncrypted : POST /media/encrypted (multipart, champ `file`). Destiné
// aux pièces jointes de messagerie E2EE : le corps est un BLOB CHIFFRÉ côté
// client (octets aléatoires) → aucune détection MIME possible ni souhaitable.
// On valide donc uniquement la taille (cap vidéo, le plus large) et on stocke
// en `application/octet-stream`. La vraie nature (image/vidéo, nom, nonce) vit
// dans l'enveloppe chiffrée du message, jamais ici → serveur aveugle.
func (h *MediaHandler) UploadEncrypted(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "non authentifié"})
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "champ `file` manquant"})
		return
	}
	if fileHeader.Size > h.maxVideoBytes {
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
		c.JSON(http.StatusBadGateway, gin.H{"error": "échec du stockage"})
		return
	}

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
func (h *MediaHandler) Download(c *gin.Context) {
	id := c.Param("id")

	obj, info, err := h.store.Open(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "média introuvable"})
			return
		}
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
		c.JSON(http.StatusForbidden, gin.H{"error": "média non détenu"})
		return
	}

	if err := h.store.Remove(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "suppression impossible"})
		return
	}
	c.Status(http.StatusNoContent)
}

// randomID génère un identifiant opaque non devinable (128 bits → 32 hex).
func randomID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
