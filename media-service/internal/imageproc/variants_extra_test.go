package imageproc

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func encodedImages(t *testing.T) ([]byte, []byte, []byte) {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 3, 2))
	img.Set(0, 0, color.NRGBA{R: 255, A: 100})
	var j, p, g bytes.Buffer
	_ = jpeg.Encode(&j, img, nil)
	_ = png.Encode(&p, img)
	if err := gif.Encode(&g, img, nil); err != nil {
		t.Fatal(err)
	}
	return j.Bytes(), p.Bytes(), g.Bytes()
}
func TestImageHelpersAndFormats(t *testing.T) {
	j, p, g := encodedImages(t)
	for _, tc := range []struct {
		data []byte
		mime string
	}{{j, "image/jpeg"}, {p, "image/png"}, {g, "image/gif"}, {p, "application/octet-stream"}} {
		if _, err := decode(tc.data, tc.mime); err != nil {
			t.Fatal(tc.mime, err)
		}
	}
	if _, err := decode([]byte("bad"), "image/png"); err == nil {
		t.Fatal("decode")
	}
	if _, _, err := generateWithStdlib([]byte("bad"), "image/png"); err == nil {
		t.Fatal("stdlib decode")
	}
	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	if b, m, err := encode(img, true, 80); err != nil || m != "image/png" || len(b) == 0 {
		t.Fatal(m, err)
	}
	if b, m, err := encode(img, false, 80); err != nil || m != "image/jpeg" || len(b) == 0 {
		t.Fatal(m, err)
	}
	if w, h := fitInside(100, 50, 200); w != 100 || h != 50 {
		t.Fatal(w, h)
	}
	if w, h := fitInside(100, 50, 20); w != 20 || h != 10 {
		t.Fatal(w, h)
	}
	if w, h := fitInside(50, 100, 20); w != 10 || h != 20 {
		t.Fatal(w, h)
	}
	if clampInt(-1, 0, 2) != 0 || clampInt(4, 0, 2) != 2 || clampInt(1, 0, 2) != 1 || max(2, 1) != 2 || max(1, 2) != 2 {
		t.Fatal("math")
	}
	e := errors.New("boom")
	if errorsWithOutput(e, nil) != e {
		t.Fatal("plain")
	}
	wrapped := errorsWithOutput(e, []byte(" details \n"))
	if wrapped.Error() != "boom: details" || !errors.Is(wrapped, e) {
		t.Fatal(wrapped)
	}
}

func TestSharpFailureModes(t *testing.T) {
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	empty := t.TempDir()
	if err = os.Chdir(empty); err != nil {
		t.Fatal(err)
	}
	if _, err = findSharpScript(); !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
	if _, _, err = generateWithSharp([]byte("x")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
	_ = os.Chdir(old)
	dir := t.TempDir()
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	for _, tc := range []struct{ name, body string }{{"missing manifest", `#!/bin/sh
exit 0`}, {"invalid manifest", `#!/bin/sh
mkdir -p "$3"; printf bad > "$3/manifest.json"`}, {"missing variant", `#!/bin/sh
mkdir -p "$3"; printf '{"width":1,"height":1,"variants":[{"name":"x","path":"/missing"}]}' > "$3/manifest.json"`}} {
		t.Run(tc.name, func(t *testing.T) {
			if err := os.WriteFile(filepath.Join(dir, "node"), []byte(tc.body), 0755); err != nil {
				t.Fatal(err)
			}
			if _, _, err := generateWithSharp([]byte("x")); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestSharpFilesystemFailures(t *testing.T) {
	t.Run("working directory removed", func(t *testing.T) {
		old, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		dir := t.TempDir()
		if err := os.Chdir(dir); err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(dir); err != nil {
			_ = os.Chdir(old)
			t.Fatal(err)
		}
		_, gotErr := findSharpScript()
		if err := os.Chdir(old); err != nil {
			t.Fatal(err)
		}
		if gotErr == nil {
			t.Fatal("findSharpScript devrait échouer sans répertoire courant")
		}
	})

	t.Run("temporary directory unavailable", func(t *testing.T) {
		t.Setenv("TMPDIR", filepath.Join(t.TempDir(), "missing"))
		if _, _, err := generateWithSharp([]byte("input")); err == nil {
			t.Fatal("generateWithSharp devrait échouer si TMPDIR est indisponible")
		}
	})
}

func TestGenerateWithSharpUsingFakeNode(t *testing.T) {
	dir := t.TempDir()
	node := filepath.Join(dir, "node")
	script := `#!/bin/sh
mkdir -p "$3"
printf variant > "$3/thumb.webp"
printf '{"width":10,"height":5,"variants":[{"name":"thumb","path":"%s/thumb.webp","mime":"image/webp","size":7,"width":10,"height":5}]}' "$3" > "$3/manifest.json"
`
	if err := os.WriteFile(node, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	info, v, err := generateWithSharp([]byte("input"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Width != 10 || len(v) != 1 || string(v[0].Bytes) != "variant" {
		t.Fatalf("%#v %#v", info, v)
	}
	info, v, err = GenerateVariants([]byte("input"), "image/jpeg")
	if err != nil || info.Width != 10 || len(v) != 1 {
		t.Fatalf("GenerateVariants: %#v %#v %v", info, v, err)
	}
}
