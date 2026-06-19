package handler

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

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
	giphyAPIKey   string
	httpClient    *http.Client
}

// NewMediaHandler construit le handler avec ses caps de taille (appliqués aux
// utilisateurs non-admin ; les admins bypassent, cf. Upload/UploadEncrypted).
func NewMediaHandler(store *storage.Store, cfg *config.Config) *MediaHandler {
	return &MediaHandler{
		store:         store,
		maxImageBytes: cfg.MaxImageBytes,
		maxVideoBytes: cfg.MaxVideoBytes,
		maxBlobBytes:  cfg.MaxBlobBytes,
		giphyAPIKey:   cfg.GiphyAPIKey,
		httpClient:    &http.Client{Timeout: 15 * time.Second},
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

// giphyResponse : réponse normalisée exposée au front. `url` est l'URL GIF
// externe à stocker dans un post ; `preview_url` sert à la grille du picker.
type giphyResponse struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	URL        string `json:"url"`
	PreviewURL string `json:"preview_url"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
}

type giphyListResponse struct {
	Data []giphyResponse `json:"data"`
}

// captureRequest : corps de POST /gifs/capture. `url` doit pointer un CDN GIPHY.
type captureRequest struct {
	URL string `json:"url"`
}

// giphyHosts : allowlist des hôtes CDN GIPHY autorisés à la capture. Borne
// stricte anti-SSRF : la capture ne fetch QUE ces hôtes, jamais une URL
// arbitraire fournie par le client.
var giphyHosts = map[string]bool{
	"media.giphy.com":  true,
	"media0.giphy.com": true,
	"media1.giphy.com": true,
	"media2.giphy.com": true,
	"media3.giphy.com": true,
	"media4.giphy.com": true,
	"i.giphy.com":      true,
}

type giphyAPIResponse struct {
	Data []struct {
		ID     string `json:"id"`
		Title  string `json:"title"`
		Images struct {
			Original struct {
				URL    string `json:"url"`
				Width  string `json:"width"`
				Height string `json:"height"`
			} `json:"original"`
			Downsized struct {
				URL    string `json:"url"`
				Width  string `json:"width"`
				Height string `json:"height"`
			} `json:"downsized"`
			FixedWidth struct {
				URL    string `json:"url"`
				Width  string `json:"width"`
				Height string `json:"height"`
			} `json:"fixed_width"`
		} `json:"images"`
	} `json:"data"`
}

// SearchGiphy : GET /gifs/search — proxy serveur vers GIPHY. Si `q` est
// vide, renvoie les tendances GIPHY pour ouvrir le picker déjà rempli.
// @Summary     Rechercher des GIFs via GIPHY
// @Tags        media
// @Produce     json
// @Security    BearerAuth
// @Param       q query string false "Recherche GIF (vide = tendances)"
// @Param       limit query int false "Nombre de GIFs (1-50, défaut 20)"
// @Success     200 {object} handler.giphyListResponse
// @Failure     401 {object} map[string]string
// @Failure     503 {object} map[string]string "GIPHY non configuré ou indisponible"
// @Router      /gifs/search [get]
func (h *MediaHandler) SearchGiphy(c *gin.Context) {
	q := strings.TrimSpace(c.Query("q"))
	limit := clampLimit(c.Query("limit"), 20, 1, 50)

	if h.giphyAPIKey == "" {
		c.JSON(http.StatusOK, giphyListResponse{Data: fallbackGifs(q, limit)})
		return
	}
	if isPublicBetaGiphyKey(h.giphyAPIKey) {
		c.JSON(http.StatusOK, giphyListResponse{Data: fallbackGifs(q, limit)})
		return
	}

	endpoint := "https://api.giphy.com/v1/gifs/trending"
	params := url.Values{}
	params.Set("api_key", h.giphyAPIKey)
	params.Set("limit", strconv.Itoa(limit))
	params.Set("rating", "pg-13")
	if q != "" {
		endpoint = "https://api.giphy.com/v1/gifs/search"
		params.Set("q", q)
		params.Set("lang", "fr")
	}

	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, endpoint+"?"+params.Encode(), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "requête giphy invalide"})
		return
	}

	res, err := h.httpClient.Do(req)
	if err != nil {
		logging.FromGin(c).Warn("giphy indisponible", "error", err)
		c.JSON(http.StatusOK, giphyListResponse{Data: fallbackGifs(q, limit)})
		return
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		logging.FromGin(c).Warn("giphy a refusé la requête", "status", res.StatusCode)
		c.JSON(http.StatusOK, giphyListResponse{Data: fallbackGifs(q, limit)})
		return
	}

	var body giphyAPIResponse
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		c.JSON(http.StatusOK, giphyListResponse{Data: fallbackGifs(q, limit)})
		return
	}

	gifs := make([]giphyResponse, 0, len(body.Data))
	for _, item := range body.Data {
		gifURL := firstNonEmpty(item.Images.Downsized.URL, item.Images.Original.URL, item.Images.FixedWidth.URL)
		previewURL := firstNonEmpty(item.Images.FixedWidth.URL, item.Images.Downsized.URL, item.Images.Original.URL)
		if gifURL == "" || previewURL == "" {
			continue
		}
		width := atoiDefault(firstNonEmpty(item.Images.Downsized.Width, item.Images.Original.Width, item.Images.FixedWidth.Width))
		height := atoiDefault(firstNonEmpty(item.Images.Downsized.Height, item.Images.Original.Height, item.Images.FixedWidth.Height))
		gifs = append(gifs, giphyResponse{
			ID:         item.ID,
			Title:      item.Title,
			URL:        gifURL,
			PreviewURL: previewURL,
			Width:      width,
			Height:     height,
		})
	}

	if len(gifs) == 0 {
		gifs = fallbackGifs(q, limit)
	}
	c.JSON(http.StatusOK, giphyListResponse{Data: gifs})
}

// CaptureGiphy : POST /gifs/capture — télécharge le GIF choisi depuis le CDN
// GIPHY et le range dans MinIO (comme un upload), pour le servir ensuite via la
// gateway (`/media/<id>`) plutôt qu'en hotlink externe. Évite les 403/404 du CDN
// giphy et reste compatible CSP `default-src 'self'`. L'URL est bornée à la
// liste d'hôtes GIPHY (anti-SSRF) ; le propriétaire est l'utilisateur capturant.
// @Summary     Capturer un GIF GIPHY dans le stockage média
// @Tags        media
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body body handler.captureRequest true "URL du GIF GIPHY"
// @Success     201 {object} handler.uploadResponse
// @Failure     400 {object} map[string]string
// @Failure     401 {object} map[string]string
// @Failure     413 {object} map[string]string "GIF trop volumineux"
// @Failure     415 {object} map[string]string "Type de média non supporté"
// @Failure     502 {object} map[string]string "GIPHY indisponible ou stockage en échec"
// @Router      /gifs/capture [post]
func (h *MediaHandler) CaptureGiphy(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "non authentifié"})
		return
	}

	var req captureRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.URL) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "champ `url` manquant"})
		return
	}

	parsed, err := url.Parse(strings.TrimSpace(req.URL))
	if err != nil || parsed.Scheme != "https" || !giphyHosts[strings.ToLower(parsed.Hostname())] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "url giphy invalide"})
		return
	}

	greq, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, parsed.String(), nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "url giphy invalide"})
		return
	}
	res, err := h.httpClient.Do(greq)
	if err != nil {
		logging.FromGin(c).Warn("capture giphy : fetch échoué", "error", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "giphy indisponible"})
		return
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		logging.FromGin(c).Warn("capture giphy : statut inattendu", "status", res.StatusCode)
		c.JSON(http.StatusBadGateway, gin.H{"error": "giphy indisponible"})
		return
	}

	// Lecture bornée à maxImageBytes+1 : un octet de rab suffit à détecter le
	// dépassement sans charger au-delà de la limite. La rendition `downsized`
	// servie par le picker fait normalement < 2 Mo.
	data, err := io.ReadAll(io.LimitReader(res.Body, h.maxImageBytes+1))
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "lecture giphy impossible"})
		return
	}
	if int64(len(data)) > h.maxImageBytes {
		logging.FromGin(c).Warn("capture giphy refusée : trop volumineux", "size", len(data))
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": validate.ErrTooLarge.Error()})
		return
	}

	head := data
	if len(head) > sniffLen {
		head = head[:sniffLen]
	}
	mime, kind, err := validate.Detect(head)
	if err != nil || kind != validate.KindImage {
		logging.FromGin(c).Warn("capture giphy refusée : type non supporté", "mime", mime, "error", err)
		c.JSON(http.StatusUnsupportedMediaType, gin.H{"error": "type de média non supporté"})
		return
	}

	id, err := randomID()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "génération d'identifiant impossible"})
		return
	}

	if err := h.store.Put(c.Request.Context(), id, bytes.NewReader(data), int64(len(data)), mime, claims.UserID); err != nil {
		logging.FromGin(c).Error("échec stockage MinIO (capture giphy)", "error", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "échec du stockage"})
		return
	}

	logging.FromGin(c).Info("gif giphy capturé", "media_id", id, "mime", mime, "size", len(data))
	c.JSON(http.StatusCreated, gin.H{"data": uploadResponse{
		ID:   id,
		URL:  "/media/" + id,
		Mime: mime,
		Kind: string(kind),
		Size: int64(len(data)),
	}})
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
	// BRZ-004 : empêche le navigateur de « renifler » un type différent de celui
	// déclaré (anti MIME-confusion). Les blobs chiffrés (octet-stream) sont en
	// outre forcés en téléchargement plutôt qu'affichés inline.
	header.Set("X-Content-Type-Options", "nosniff")
	if info.ContentType == "application/octet-stream" {
		header.Set("Content-Disposition", "attachment")
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

func clampLimit(raw string, fallback, minValue, maxValue int) int {
	n, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	if n < minValue {
		return minValue
	}
	if n > maxValue {
		return maxValue
	}
	return n
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func atoiDefault(raw string) int {
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}
	return n
}

func isPublicBetaGiphyKey(key string) bool {
	return strings.TrimSpace(key) == "dc6zaTOxFJmzC"
}

var fallbackGifCatalog = []giphyResponse{
	{
		ID:         "fallback-cat",
		Title:      "Cat typing",
		URL:        "https://media.giphy.com/media/JIX9t2j0ZTN9S/giphy.gif",
		PreviewURL: "https://media.giphy.com/media/JIX9t2j0ZTN9S/200.gif",
		Width:      480,
		Height:     360,
	},
	{
		ID:         "fallback-happy",
		Title:      "Happy dance",
		URL:        "https://media.giphy.com/media/26u4cqiYI30juCOGY/giphy.gif",
		PreviewURL: "https://media.giphy.com/media/26u4cqiYI30juCOGY/200.gif",
		Width:      480,
		Height:     270,
	},
	{
		ID:         "fallback-excited",
		Title:      "Excited",
		URL:        "https://media.giphy.com/media/l0MYt5jPR6QX5pnqM/giphy.gif",
		PreviewURL: "https://media.giphy.com/media/l0MYt5jPR6QX5pnqM/200.gif",
		Width:      480,
		Height:     360,
	},
	{
		ID:         "fallback-thumbs-up",
		Title:      "Thumbs up",
		URL:        "https://media.giphy.com/media/111ebonMs90YLu/giphy.gif",
		PreviewURL: "https://media.giphy.com/media/111ebonMs90YLu/200.gif",
		Width:      400,
		Height:     300,
	},
	{
		ID:         "fallback-hello",
		Title:      "Hello wave",
		URL:        "https://media.giphy.com/media/ASd0Ukj0y3qMM/giphy.gif",
		PreviewURL: "https://media.giphy.com/media/ASd0Ukj0y3qMM/200.gif",
		Width:      480,
		Height:     360,
	},
	{
		ID:         "fallback-laugh",
		Title:      "Laugh",
		URL:        "https://media.giphy.com/media/10JhviFuU2gWD6/giphy.gif",
		PreviewURL: "https://media.giphy.com/media/10JhviFuU2gWD6/200.gif",
		Width:      480,
		Height:     360,
	},
	{
		ID:         "fallback-nope",
		Title:      "Nope",
		URL:        "https://media.giphy.com/media/3o7TKwmnDgQb5jemjK/giphy.gif",
		PreviewURL: "https://media.giphy.com/media/3o7TKwmnDgQb5jemjK/200.gif",
		Width:      480,
		Height:     270,
	},
	{
		ID:         "fallback-mind-blown",
		Title:      "Mind blown",
		URL:        "https://media.giphy.com/media/Um3ljJl8jrnHy/giphy.gif",
		PreviewURL: "https://media.giphy.com/media/Um3ljJl8jrnHy/200.gif",
		Width:      480,
		Height:     360,
	},
	{
		ID:         "fallback-party",
		Title:      "Party celebration",
		URL:        "https://media.giphy.com/media/3KC2jD2QcBOSc/giphy.gif",
		PreviewURL: "https://media.giphy.com/media/3KC2jD2QcBOSc/200.gif",
		Width:      480,
		Height:     270,
	},
	{
		ID:         "fallback-clap",
		Title:      "Clap applause",
		URL:        "https://media.giphy.com/media/l3q2XhfQ8oCkm1Ts4/giphy.gif",
		PreviewURL: "https://media.giphy.com/media/l3q2XhfQ8oCkm1Ts4/200.gif",
		Width:      480,
		Height:     360,
	},
	{
		ID:         "fallback-shrug",
		Title:      "Shrug",
		URL:        "https://media.giphy.com/media/3o7btPCcdNniyf0ArS/giphy.gif",
		PreviewURL: "https://media.giphy.com/media/3o7btPCcdNniyf0ArS/200.gif",
		Width:      480,
		Height:     270,
	},
	{
		ID:         "fallback-facepalm",
		Title:      "Facepalm",
		URL:        "https://media.giphy.com/media/3og0INyCmHlNylks9O/giphy.gif",
		PreviewURL: "https://media.giphy.com/media/3og0INyCmHlNylks9O/200.gif",
		Width:      480,
		Height:     270,
	},
	{
		ID:         "fallback-sad",
		Title:      "Sad",
		URL:        "https://media.giphy.com/media/OPU6wzx8JrHna/giphy.gif",
		PreviewURL: "https://media.giphy.com/media/OPU6wzx8JrHna/200.gif",
		Width:      480,
		Height:     360,
	},
	{
		ID:         "fallback-cry",
		Title:      "Cry",
		URL:        "https://media.giphy.com/media/d2lcHJTG5Tscg/giphy.gif",
		PreviewURL: "https://media.giphy.com/media/d2lcHJTG5Tscg/200.gif",
		Width:      480,
		Height:     360,
	},
	{
		ID:         "fallback-love",
		Title:      "Love hearts",
		URL:        "https://media.giphy.com/media/26FLdmIp6wJr91JAI/giphy.gif",
		PreviewURL: "https://media.giphy.com/media/26FLdmIp6wJr91JAI/200.gif",
		Width:      480,
		Height:     360,
	},
	{
		ID:         "fallback-ok",
		Title:      "Ok good",
		URL:        "https://media.giphy.com/media/xT0BKL21U5nnlW4m6k/giphy.gif",
		PreviewURL: "https://media.giphy.com/media/xT0BKL21U5nnlW4m6k/200.gif",
		Width:      480,
		Height:     270,
	},
	{
		ID:         "fallback-done",
		Title:      "Done success",
		URL:        "https://media.giphy.com/media/11sBLVxNs7v6WA/giphy.gif",
		PreviewURL: "https://media.giphy.com/media/11sBLVxNs7v6WA/200.gif",
		Width:      480,
		Height:     360,
	},
	{
		ID:         "fallback-coffee",
		Title:      "Coffee",
		URL:        "https://media.giphy.com/media/687qS11pXwjCM/giphy.gif",
		PreviewURL: "https://media.giphy.com/media/687qS11pXwjCM/200.gif",
		Width:      480,
		Height:     360,
	},
	{
		ID:         "fallback-working",
		Title:      "Working typing",
		URL:        "https://media.giphy.com/media/13HgwGsXF0aiGY/giphy.gif",
		PreviewURL: "https://media.giphy.com/media/13HgwGsXF0aiGY/200.gif",
		Width:      480,
		Height:     360,
	},
	{
		ID:         "fallback-dance",
		Title:      "Dance",
		URL:        "https://media.giphy.com/media/GeimqsH0TLDt4tScGw/giphy.gif",
		PreviewURL: "https://media.giphy.com/media/GeimqsH0TLDt4tScGw/200.gif",
		Width:      480,
		Height:     480,
	},
	{
		ID:         "fallback-yay",
		Title:      "Yay",
		URL:        "https://media.giphy.com/media/artj92V8o75VPL7AeQ/giphy.gif",
		PreviewURL: "https://media.giphy.com/media/artj92V8o75VPL7AeQ/200.gif",
		Width:      480,
		Height:     480,
	},
	{
		ID:         "fallback-fire",
		Title:      "Fire",
		URL:        "https://media.giphy.com/media/yr7n0u3qzO9nG/giphy.gif",
		PreviewURL: "https://media.giphy.com/media/yr7n0u3qzO9nG/200.gif",
		Width:      480,
		Height:     360,
	},
	{
		ID:         "fallback-popcorn",
		Title:      "Popcorn",
		URL:        "https://media.giphy.com/media/2UvAUplPi4ESnKa3W0/giphy.gif",
		PreviewURL: "https://media.giphy.com/media/2UvAUplPi4ESnKa3W0/200.gif",
		Width:      480,
		Height:     270,
	},
	{
		ID:         "fallback-please",
		Title:      "Please",
		URL:        "https://media.giphy.com/media/CT5Ye7uVJLFtu/giphy.gif",
		PreviewURL: "https://media.giphy.com/media/CT5Ye7uVJLFtu/200.gif",
		Width:      480,
		Height:     360,
	},
	{
		ID:         "fallback-thinking",
		Title:      "Thinking",
		URL:        "https://media.giphy.com/media/a5viI92PAF89q/giphy.gif",
		PreviewURL: "https://media.giphy.com/media/a5viI92PAF89q/200.gif",
		Width:      480,
		Height:     360,
	},
	{
		ID:         "fallback-sparkle",
		Title:      "Sparkle magic",
		URL:        "https://media.giphy.com/media/3o7aD2saalBwwftBIY/giphy.gif",
		PreviewURL: "https://media.giphy.com/media/3o7aD2saalBwwftBIY/200.gif",
		Width:      480,
		Height:     270,
	},
}

func fallbackGifs(query string, limit int) []giphyResponse {
	q := strings.ToLower(strings.TrimSpace(query))
	out := make([]giphyResponse, 0, min(limit, len(fallbackGifCatalog)))
	for _, gif := range fallbackGifCatalog {
		if q != "" && !strings.Contains(strings.ToLower(gif.Title), q) {
			continue
		}
		out = append(out, gif)
		if len(out) >= limit {
			return out
		}
	}
	if len(out) > 0 || q == "" {
		return out
	}
	for _, gif := range fallbackGifCatalog {
		out = append(out, gif)
		if len(out) >= limit {
			break
		}
	}
	return out
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
