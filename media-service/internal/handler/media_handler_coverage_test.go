package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/gif"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"

	"github.com/webdad/media-service/internal/config"
	"github.com/webdad/media-service/internal/middleware"
	"github.com/webdad/media-service/internal/storage"
)

type memoryObject struct{ *bytes.Reader }

func (memoryObject) Close() error { return nil }

type fakeMediaStore struct {
	data                                          map[string][]byte
	info                                          map[string]minio.ObjectInfo
	putErr, openErr, statErr, removeErr, purgeErr error
	variantErr                                    error
	purgeCount                                    int
}

func newFakeMediaStore() *fakeMediaStore {
	return &fakeMediaStore{data: map[string][]byte{}, info: map[string]minio.ObjectInfo{}}
}
func (s *fakeMediaStore) Put(_ context.Context, id string, r io.Reader, _ int64, mime, owner string) error {
	if s.putErr != nil {
		return s.putErr
	}
	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	s.data[id] = b
	s.info[id] = minio.ObjectInfo{Key: id, Size: int64(len(b)), ContentType: mime, LastModified: time.Unix(10, 0), ETag: "etag", Metadata: http.Header{"X-Amz-Meta-Owner-Id": []string{owner}}}
	return nil
}
func (s *fakeMediaStore) PutVariant(ctx context.Context, id, variant string, r io.Reader, size int64, mime, owner string) error {
	if s.variantErr != nil {
		return s.variantErr
	}
	return s.Put(ctx, storage.VariantKey(id, variant), r, size, mime, owner)
}
func (s *fakeMediaStore) Open(_ context.Context, id string) (storage.Object, minio.ObjectInfo, error) {
	if s.openErr != nil {
		return nil, minio.ObjectInfo{}, s.openErr
	}
	b, ok := s.data[id]
	if !ok {
		return nil, minio.ObjectInfo{}, storage.ErrNotFound
	}
	return memoryObject{bytes.NewReader(b)}, s.info[id], nil
}
func (s *fakeMediaStore) Stat(_ context.Context, id string) (minio.ObjectInfo, error) {
	if s.statErr != nil {
		return minio.ObjectInfo{}, s.statErr
	}
	i, ok := s.info[id]
	if !ok {
		return minio.ObjectInfo{}, storage.ErrNotFound
	}
	return i, nil
}
func (s *fakeMediaStore) RemoveMediaSet(_ context.Context, id string) error {
	if s.removeErr != nil {
		return s.removeErr
	}
	delete(s.data, id)
	return nil
}
func (s *fakeMediaStore) RemoveByOwner(context.Context, string) (int, error) {
	return s.purgeCount, s.purgeErr
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type errorReader struct{}

func (errorReader) Read([]byte) (int, error) { return 0, errors.New("read") }

func coverageHandler(store mediaStore, key string) *MediaHandler {
	return NewMediaHandler(store, &config.Config{MaxImageBytes: 1 << 20, MaxVideoBytes: 2 << 20, MaxBlobBytes: 1 << 20, GiphyAPIKey: key})
}
func coverageRouter(h *MediaHandler, claims *middleware.Claims) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	if claims != nil {
		r.Use(func(c *gin.Context) { c.Set("claims", claims); c.Next() })
	}
	r.GET("/gifs", h.SearchGiphy)
	r.POST("/capture", h.CaptureGiphy)
	r.POST("/upload", h.Upload)
	r.POST("/encrypted", h.UploadEncrypted)
	r.GET("/media/:id", h.Download)
	r.GET("/media/:id/:variant", h.DownloadVariant)
	r.DELETE("/media/:id", h.Delete)
	r.DELETE("/owners/:id", h.PurgeByOwner)
	return r
}
func perform(r http.Handler, method, path string, body io.Reader, contentType string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, body)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	r.ServeHTTP(w, req)
	return w
}
func tinyGIF(t *testing.T) []byte {
	t.Helper()
	var b bytes.Buffer
	img := image.NewPaletted(image.Rect(0, 0, 2, 2), color.Palette{color.Black, color.White})
	if err := gif.Encode(&b, img, nil); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestSearchGiphyCoverage(t *testing.T) {
	t.Run("fallbacks and helpers", func(t *testing.T) {
		h := coverageHandler(newFakeMediaStore(), "")
		r := coverageRouter(h, nil)
		for _, path := range []string{"/gifs?limit=2", "/gifs?q=cat&limit=0", "/gifs?q=absent&limit=999", "/gifs?q=sparkle&limit=50"} {
			if w := perform(r, "GET", path, nil, ""); w.Code != 200 {
				t.Fatal(w.Code)
			}
		}
		h.giphyAPIKey = "dc6zaTOxFJmzC"
		perform(r, "GET", "/gifs", nil, "")
		if !validVariant("thumb") || validVariant("bad") || clampLimit("x", 4, 1, 5) != 4 || clampLimit("0", 4, 1, 5) != 1 || clampLimit("9", 4, 1, 5) != 5 || clampLimit("3", 4, 1, 5) != 3 {
			t.Fatal("helpers")
		}
		if firstNonEmpty("", "x") != "x" || firstNonEmpty("") != "" || atoiDefault("x") != 0 || atoiDefault("12") != 12 || !isPublicBetaGiphyKey(" dc6zaTOxFJmzC ") {
			t.Fatal("helpers")
		}
	})
	t.Run("api outcomes", func(t *testing.T) {
		cases := []struct {
			name         string
			status       int
			body         string
			transportErr bool
		}{
			{"success", 200, `{"data":[{"id":"1","title":"ok","images":{"original":{"url":"o","width":"12","height":"bad"},"downsized":{"url":"d"},"fixed_width":{"url":"p"}}},{"id":"skip","images":{}}]}`, false},
			{"empty", 200, `{"data":[]}`, false}, {"invalid", 200, `{`, false}, {"status", 503, ``, false}, {"network", 0, ``, true},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				h := coverageHandler(newFakeMediaStore(), "private")
				h.httpClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					if tc.transportErr {
						return nil, errors.New("offline")
					}
					if !strings.Contains(req.URL.RawQuery, "api_key=private") {
						t.Error(req.URL)
					}
					return &http.Response{StatusCode: tc.status, Body: io.NopCloser(strings.NewReader(tc.body)), Header: make(http.Header)}, nil
				})}
				r := coverageRouter(h, nil)
				if w := perform(r, "GET", "/gifs?q=hey&limit=3", nil, ""); w.Code != 200 {
					t.Fatal(w.Code)
				}
			})
		}
	})
}

func TestCaptureGiphyCoverage(t *testing.T) {
	claims := &middleware.Claims{UserID: "owner", Role: "user"}
	t.Run("validation", func(t *testing.T) {
		h := coverageHandler(newFakeMediaStore(), "")
		if w := perform(coverageRouter(h, nil), "POST", "/capture", strings.NewReader(`{"url":"https://media.giphy.com/a.gif"}`), "application/json"); w.Code != 401 {
			t.Fatal(w.Code)
		}
		r := coverageRouter(h, claims)
		for _, body := range []string{`{}`, `bad`, `{"url":"http://media.giphy.com/a"}`, `{"url":"https://evil.test/a"}`} {
			if w := perform(r, "POST", "/capture", strings.NewReader(body), "application/json"); w.Code != 400 {
				t.Fatalf("%s: %d", body, w.Code)
			}
		}
	})
	for _, tc := range []struct {
		name   string
		status int
		data   []byte
		err    error
		putErr error
		want   int
	}{
		{"network", 0, nil, errors.New("x"), nil, 502}, {"remote", 404, nil, nil, nil, 502}, {"large", 200, make([]byte, (1<<20)+1), nil, nil, 413}, {"mime", 200, []byte("plain"), nil, nil, 415}, {"store", 200, tinyGIF(t), nil, errors.New("store"), 502}, {"ok", 200, tinyGIF(t), nil, nil, 201},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := newFakeMediaStore()
			s.putErr = tc.putErr
			h := coverageHandler(s, "")
			h.httpClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				if tc.err != nil {
					return nil, tc.err
				}
				return &http.Response{StatusCode: tc.status, Body: io.NopCloser(bytes.NewReader(tc.data)), Header: make(http.Header)}, nil
			})}
			w := perform(coverageRouter(h, claims), "POST", "/capture", strings.NewReader(`{"url":"https://media.giphy.com/a.gif"}`), "application/json")
			if w.Code != tc.want {
				t.Fatalf("%d: %s", w.Code, w.Body.String())
			}
		})
	}
	t.Run("read error", func(t *testing.T) {
		h := coverageHandler(newFakeMediaStore(), "")
		h.httpClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(errorReader{}), Header: make(http.Header)}, nil
		})}
		w := perform(coverageRouter(h, claims), "POST", "/capture", strings.NewReader(`{"url":"https://media.giphy.com/a.gif"}`), "application/json")
		if w.Code != 502 {
			t.Fatal(w.Code)
		}
	})
	t.Run("long image and random failure", func(t *testing.T) {
		old := randomRead
		randomRead = func([]byte) (int, error) { return 0, errors.New("random") }
		defer func() { randomRead = old }()
		data := append(tinyGIF(t), make([]byte, sniffLen)...)
		h := coverageHandler(newFakeMediaStore(), "")
		h.httpClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(data)), Header: make(http.Header)}, nil
		})}
		w := perform(coverageRouter(h, claims), "POST", "/capture", strings.NewReader(`{"url":"https://media.giphy.com/a.gif"}`), "application/json")
		if w.Code != 500 {
			t.Fatal(w.Code)
		}
	})
}

func TestMediaStoreBackedHandlers(t *testing.T) {
	claims := &middleware.Claims{UserID: "owner", Role: "user"}
	t.Run("encrypted", func(t *testing.T) {
		for _, putErr := range []error{nil, errors.New("x")} {
			s := newFakeMediaStore()
			s.putErr = putErr
			h := coverageHandler(s, "")
			body, ct := buildMultipartFile(t, "file", "x.bin", []byte("cipher"))
			w := perform(coverageRouter(h, claims), "POST", "/encrypted", body, ct)
			want := 201
			if putErr != nil {
				want = 502
			}
			if w.Code != want {
				t.Fatal(w.Code)
			}
		}
	})
	t.Run("upload video", func(t *testing.T) {
		s := newFakeMediaStore()
		h := coverageHandler(s, "")
		mp4 := append([]byte{0, 0, 0, 24, 'f', 't', 'y', 'p', 'i', 's', 'o', 'm'}, make([]byte, 30)...)
		body, ct := buildMultipartFile(t, "file", "x.mp4", mp4)
		w := perform(coverageRouter(h, claims), "POST", "/upload", body, ct)
		if w.Code != 201 {
			t.Fatalf("%d %s", w.Code, w.Body.String())
		}
	})
	t.Run("upload image and variants", func(t *testing.T) {
		s := newFakeMediaStore()
		h := coverageHandler(s, "")
		body, ct := buildMultipartFile(t, "file", "x.gif", tinyGIF(t))
		w := perform(coverageRouter(h, claims), "POST", "/upload", body, ct)
		if w.Code != http.StatusCreated {
			t.Fatalf("%d %s", w.Code, w.Body.String())
		}
		if len(s.data) < 2 {
			t.Fatalf("variantes absentes: %v", s.data)
		}
	})
	t.Run("upload image transform and variant errors", func(t *testing.T) {
		s := newFakeMediaStore()
		h := coverageHandler(s, "")
		body, ct := buildMultipartFile(t, "file", "bad.jpg", minimalJPEG())
		if w := perform(coverageRouter(h, claims), "POST", "/upload", body, ct); w.Code != 201 {
			t.Fatal(w.Code)
		}
		s = newFakeMediaStore()
		s.variantErr = errors.New("variant")
		h = coverageHandler(s, "")
		body, ct = buildMultipartFile(t, "file", "x.gif", tinyGIF(t))
		if w := perform(coverageRouter(h, claims), "POST", "/upload", body, ct); w.Code != 201 {
			t.Fatal(w.Code)
		}
	})
	t.Run("random failures", func(t *testing.T) {
		old := randomRead
		randomRead = func([]byte) (int, error) { return 0, errors.New("random") }
		defer func() { randomRead = old }()
		h := coverageHandler(newFakeMediaStore(), "")
		body, ct := buildMultipartFile(t, "file", "x.gif", tinyGIF(t))
		if w := perform(coverageRouter(h, claims), "POST", "/upload", body, ct); w.Code != 500 {
			t.Fatal(w.Code)
		}
		body, ct = buildMultipartFile(t, "file", "x.bin", []byte("x"))
		if w := perform(coverageRouter(h, claims), "POST", "/encrypted", body, ct); w.Code != 500 {
			t.Fatal(w.Code)
		}
		if _, err := randomID(); err == nil {
			t.Fatal("randomID")
		}
	})
	t.Run("upload store error", func(t *testing.T) {
		s := newFakeMediaStore()
		s.putErr = errors.New("x")
		h := coverageHandler(s, "")
		body, ct := buildMultipartFile(t, "file", "x.gif", tinyGIF(t))
		if w := perform(coverageRouter(h, claims), "POST", "/upload", body, ct); w.Code != 502 {
			t.Fatal(w.Code)
		}
	})
	t.Run("download", func(t *testing.T) {
		s := newFakeMediaStore()
		_ = s.Put(context.Background(), "x", strings.NewReader("hello"), 5, "application/octet-stream", "owner")
		h := coverageHandler(s, "")
		r := coverageRouter(h, claims)
		w := perform(r, "GET", "/media/x", nil, "")
		if w.Code != 200 || w.Body.String() != "hello" || w.Header().Get("Content-Disposition") != "attachment" {
			t.Fatalf("%d %q", w.Code, w.Body.String())
		}
		if w := perform(r, "GET", "/media/x/bad", nil, ""); w.Code != 404 {
			t.Fatal(w.Code)
		}
		if w := perform(r, "GET", "/media/x/thumb", nil, ""); w.Code != 404 {
			t.Fatal(w.Code)
		}
		s.openErr = errors.New("down")
		if w := perform(r, "GET", "/media/x", nil, ""); w.Code != 502 {
			t.Fatal(w.Code)
		}
		s.openErr = storage.ErrNotFound
		if w := perform(r, "GET", "/media/x", nil, ""); w.Code != 404 {
			t.Fatal(w.Code)
		}
	})
	t.Run("delete", func(t *testing.T) {
		tests := []struct {
			name               string
			claims             *middleware.Claims
			statErr, removeErr error
			owner              string
			want               int
		}{{"missing", claims, storage.ErrNotFound, nil, "", 404}, {"stat", claims, errors.New("x"), nil, "", 502}, {"forbidden", claims, nil, nil, "other", 403}, {"remove", claims, nil, errors.New("x"), "owner", 502}, {"owner", claims, nil, nil, "owner", 204}, {"admin", &middleware.Claims{UserID: "admin", Role: "admin"}, nil, nil, "other", 204}}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				s := newFakeMediaStore()
				s.statErr = tc.statErr
				s.removeErr = tc.removeErr
				s.info["x"] = minio.ObjectInfo{Metadata: http.Header{"X-Amz-Meta-Owner-Id": []string{tc.owner}}}
				w := perform(coverageRouter(coverageHandler(s, ""), tc.claims), "DELETE", "/media/x", nil, "")
				if w.Code != tc.want {
					t.Fatal(w.Code)
				}
			})
		}
	})
	t.Run("purge", func(t *testing.T) {
		s := newFakeMediaStore()
		s.purgeCount = 3
		h := coverageHandler(s, "")
		w := perform(coverageRouter(h, claims), "DELETE", "/owners/o", nil, "")
		if w.Code != 200 {
			t.Fatal(w.Code)
		}
		var v map[string]any
		if json.Unmarshal(w.Body.Bytes(), &v) != nil {
			t.Fatal("json")
		}
		s.purgeErr = errors.New("x")
		if w := perform(coverageRouter(h, claims), "DELETE", "/owners/o", nil, ""); w.Code != 502 {
			t.Fatal(w.Code)
		}
	})
}
