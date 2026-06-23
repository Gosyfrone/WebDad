package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type s3Fixture struct {
	mu          sync.Mutex
	objects     map[string][]byte
	owners      map[string]string
	exists      bool
	failMethod  string
	failKey     string
	badList     bool
	statMissing string
}

func (f *s3Fixture) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.Method == f.failMethod && (f.failKey == "" || strings.HasSuffix(r.URL.Path, f.failKey)) {
		return nil, errors.New("transport failure")
	}
	w := &responseRecorder{header: make(http.Header), code: http.StatusOK}
	f.ServeHTTP(w, r)
	return &http.Response{StatusCode: w.code, Header: w.header, Body: io.NopCloser(&w.body), Request: r}, nil
}

func TestStoreFailures(t *testing.T) {
	ctx := context.Background()
	t.Run("remove original", func(t *testing.T) {
		s, f := fixtureStore(t, true)
		f.objects["x"] = []byte("x")
		f.failMethod = http.MethodDelete
		if err := s.RemoveMediaSet(ctx, "x"); err == nil {
			t.Fatal("error")
		}
	})
	t.Run("list variants", func(t *testing.T) {
		s, f := fixtureStore(t, true)
		f.objects["x"] = []byte("x")
		f.failMethod = http.MethodGet
		if err := s.RemoveMediaSet(ctx, "x"); err == nil {
			t.Fatal("error")
		}
	})
	t.Run("remove variant", func(t *testing.T) {
		s, f := fixtureStore(t, true)
		f.objects["x"] = []byte("x")
		f.objects["x/thumb"] = []byte("v")
		f.failMethod = http.MethodDelete
		f.failKey = "thumb"
		if err := s.RemoveMediaSet(ctx, "x"); err == nil {
			t.Fatal("error")
		}
	})
	t.Run("purge list", func(t *testing.T) {
		s, f := fixtureStore(t, true)
		f.failMethod = http.MethodGet
		if _, err := s.RemoveByOwner(ctx, "x"); err == nil {
			t.Fatal("error")
		}
	})
	t.Run("purge stat", func(t *testing.T) {
		s, f := fixtureStore(t, true)
		f.objects["x"] = []byte("x")
		f.failMethod = http.MethodHead
		if _, err := s.RemoveByOwner(ctx, "x"); err == nil {
			t.Fatal("error")
		}
	})
	t.Run("purge remove", func(t *testing.T) {
		s, f := fixtureStore(t, true)
		f.objects["x"] = []byte("x")
		f.owners["x"] = "owner"
		f.failMethod = http.MethodDelete
		if _, err := s.RemoveByOwner(ctx, "owner"); err == nil {
			t.Fatal("error")
		}
	})
	t.Run("open generic", func(t *testing.T) {
		s, f := fixtureStore(t, true)
		f.objects["x"] = []byte("x")
		f.failMethod = http.MethodHead
		if _, _, err := s.Open(ctx, "x"); err == nil || errors.Is(err, ErrNotFound) {
			t.Fatal(err)
		}
	})
	t.Run("bad variant list", func(t *testing.T) {
		s, f := fixtureStore(t, true)
		f.objects["x"] = []byte("x")
		f.badList = true
		if err := s.RemoveMediaSet(ctx, "x"); err == nil {
			t.Fatal("error")
		}
	})
	t.Run("owner race", func(t *testing.T) {
		s, f := fixtureStore(t, true)
		f.objects["x"] = []byte("x")
		f.statMissing = "x"
		if n, err := s.RemoveByOwner(ctx, "owner"); err != nil || n != 0 {
			t.Fatal(n, err)
		}
	})
}

func TestNew(t *testing.T) {
	original := newMinioClient
	defer func() { newMinioClient = original }()
	for _, tc := range []struct {
		name    string
		exists  bool
		fail    string
		wantErr bool
	}{{"exists", true, "", false}, {"creates", false, "", false}, {"head error", false, http.MethodHead, true}, {"create error", false, http.MethodPut, true}} {
		t.Run(tc.name, func(t *testing.T) {
			f := &s3Fixture{objects: map[string][]byte{}, owners: map[string]string{}, exists: tc.exists, failMethod: tc.fail}
			newMinioClient = func(string, *minio.Options) (*minio.Client, error) {
				return minio.New("s3.test", &minio.Options{Creds: credentials.NewStaticV4("a", "b", ""), Secure: true, Transport: f, MaxRetries: 1})
			}
			s, err := New("ignored", "a", "b", false, "bucket")
			if (err != nil) != tc.wantErr {
				t.Fatalf("store=%v err=%v", s, err)
			}
			if err == nil && !f.exists {
				t.Fatal("bucket not created")
			}
		})
	}
	newMinioClient = func(string, *minio.Options) (*minio.Client, error) { return nil, errors.New("constructor") }
	if _, err := New("x", "a", "b", false, "bucket"); err == nil {
		t.Fatal("constructor")
	}
}

type responseRecorder struct {
	header http.Header
	body   bytes.Buffer
	code   int
}

func (w *responseRecorder) Header() http.Header         { return w.header }
func (w *responseRecorder) WriteHeader(code int)        { w.code = code }
func (w *responseRecorder) Write(p []byte) (int, error) { return w.body.Write(p) }

func (f *s3Fixture) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := strings.TrimPrefix(r.URL.Path, "/bucket/")
	isBucket := strings.TrimSuffix(r.URL.Path, "/") == "/bucket"
	if isBucket && r.Method == http.MethodHead {
		if !f.exists {
			http.Error(w, "", 404)
			return
		}
		w.WriteHeader(200)
		return
	}
	if isBucket && r.Method == http.MethodPut {
		f.exists = true
		w.WriteHeader(200)
		return
	}
	if isBucket && r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "application/xml")
		if f.badList {
			_, _ = w.Write([]byte("not xml"))
			return
		}
		fmt.Fprint(w, `<?xml version="1.0"?><ListBucketResult><Name>bucket</Name><IsTruncated>false</IsTruncated>`)
		for k := range f.objects {
			fmt.Fprintf(w, "<Contents><Key>%s</Key><Size>%d</Size></Contents>", k, len(f.objects[k]))
		}
		fmt.Fprint(w, "</ListBucketResult>")
		return
	}
	switch r.Method {
	case http.MethodPut:
		b, _ := io.ReadAll(r.Body)
		f.objects[key] = b
		f.owners[key] = r.Header.Get("X-Amz-Meta-Owner-Id")
		w.Header().Set("ETag", `"etag"`)
		w.WriteHeader(200)
	case http.MethodHead:
		if key == f.statMissing {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		b, ok := f.objects[key]
		if !ok {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(404)
			return
		}
		w.Header().Set("Content-Length", fmt.Sprint(len(b)))
		w.Header().Set("Last-Modified", "Mon, 02 Jan 2006 15:04:05 GMT")
		w.Header().Set("ETag", `"etag"`)
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("ETag", `"etag"`)
		w.Header().Set("Last-Modified", "Mon, 02 Jan 2006 15:04:05 GMT")
		w.Header().Set("X-Amz-Meta-Owner-Id", f.owners[key])
		w.WriteHeader(200)
	case http.MethodGet:
		b, ok := f.objects[key]
		if !ok {
			w.WriteHeader(404)
			return
		}
		w.Header().Set("Content-Length", fmt.Sprint(len(b)))
		w.Header().Set("Last-Modified", "Mon, 02 Jan 2006 15:04:05 GMT")
		w.Header().Set("ETag", `"etag"`)
		if r.Header.Get("Range") != "" {
			w.Header().Set("Content-Range", fmt.Sprintf("bytes 0-%d/%d", len(b)-1, len(b)))
			w.WriteHeader(http.StatusPartialContent)
		}
		_, _ = w.Write(b)
	case http.MethodDelete:
		delete(f.objects, key)
		delete(f.owners, key)
		w.WriteHeader(204)
	}
}

func fixtureStore(t *testing.T, exists bool) (*Store, *s3Fixture) {
	t.Helper()
	f := &s3Fixture{objects: map[string][]byte{}, owners: map[string]string{}, exists: exists}
	client, err := minio.New("s3.test", &minio.Options{Creds: credentials.NewStaticV4("access", "secret", ""), Secure: true, Transport: f, MaxRetries: 1})
	if err != nil {
		t.Fatal(err)
	}
	return &Store{client: client, bucket: "bucket"}, f
}
func TestStoreLifecycle(t *testing.T) {
	for _, exists := range []bool{true, false} {
		t.Run(fmt.Sprint(exists), func(t *testing.T) {
			s, f := fixtureStore(t, exists)
			ctx := context.Background()
			if err := s.Put(ctx, "one", strings.NewReader("hello"), 5, "text/plain", "owner"); err != nil {
				t.Fatal(err)
			}
			if err := s.PutVariant(ctx, "one", "/thumb/", strings.NewReader("v"), 1, "text/plain", "owner"); err != nil {
				t.Fatal(err)
			}
			if VariantKey("one", "/thumb/") != "one/thumb" {
				t.Fatal("variant")
			}
			info, err := s.Stat(ctx, "one")
			if err != nil || OwnerOf(info) != "owner" {
				t.Fatalf("%v %#v", err, info.Metadata)
			}
			obj, _, err := s.Open(ctx, "one")
			if err != nil {
				t.Fatal(err)
			}
			b, readErr := io.ReadAll(obj)
			_ = obj.Close()
			if readErr != nil || !bytes.Equal(b, []byte("hello")) {
				t.Fatal(string(b), readErr)
			}
			if err := s.RemoveMediaSet(ctx, "one"); err != nil {
				t.Fatal(err)
			}
			if len(f.objects) != 0 {
				t.Fatal(f.objects)
			}
		})
	}
}
func TestRemoveByOwnerAndNotFound(t *testing.T) {
	s, f := fixtureStore(t, true)
	ctx := context.Background()
	_ = s.Put(ctx, "a", strings.NewReader("a"), 1, "text/plain", "target")
	_ = s.Put(ctx, "b", strings.NewReader("b"), 1, "text/plain", "other")
	n, err := s.RemoveByOwner(ctx, "target")
	if err != nil || n != 1 {
		t.Fatal(n, err)
	}
	if _, ok := f.objects["b"]; !ok {
		t.Fatal("other removed")
	}
	if _, err := s.Stat(ctx, "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
	if _, _, err := s.Open(ctx, "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
	if !isNotFound(minio.ErrorResponse{Code: "NoSuchKey"}) || !isNotFound(minio.ErrorResponse{StatusCode: 404}) || isNotFound(errors.New("x")) {
		t.Fatal("not found")
	}
}
