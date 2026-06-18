package imageproc

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

func TestGenerateVariantsJPEG(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 2400, 1200))
	for y := 0; y < src.Bounds().Dy(); y++ {
		for x := 0; x < src.Bounds().Dx(); x++ {
			src.SetRGBA(x, y, color.RGBA{R: uint8(x % 255), G: uint8(y % 255), B: 140, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, src, &jpeg.Options{Quality: 95}); err != nil {
		t.Fatal(err)
	}

	info, variants, err := GenerateVariants(buf.Bytes(), "image/jpeg")
	if err != nil {
		t.Fatal(err)
	}
	if info.Width != 2400 || info.Height != 1200 {
		t.Fatalf("dimensions original = %dx%d", info.Width, info.Height)
	}
	if len(variants) != 4 {
		t.Fatalf("variantes = %d, attendu 4", len(variants))
	}
	if variants[0].Name != "thumb" || variants[0].Width != 320 || variants[0].Height != 160 {
		t.Fatalf("thumb inattendue: %#v", variants[0])
	}
	if variants[len(variants)-1].Name != "large" || variants[len(variants)-1].Width != 2048 {
		t.Fatalf("large inattendue: %#v", variants[len(variants)-1])
	}
	for _, v := range variants {
		if v.Mime != "image/jpeg" && v.Mime != "image/webp" {
			t.Fatalf("%s mime = %s", v.Name, v.Mime)
		}
		if len(v.Bytes) == 0 {
			t.Fatalf("%s vide", v.Name)
		}
	}
}

func TestGenerateVariantsTransparentPNG(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 800, 400))
	src.SetNRGBA(10, 10, color.NRGBA{R: 255, A: 120})
	var buf bytes.Buffer
	if err := png.Encode(&buf, src); err != nil {
		t.Fatal(err)
	}

	_, variants, err := GenerateVariants(buf.Bytes(), "image/png")
	if err != nil {
		t.Fatal(err)
	}
	if len(variants) == 0 {
		t.Fatal("aucune variante générée")
	}
	for _, v := range variants {
		if v.Mime != "image/png" && v.Mime != "image/webp" {
			t.Fatalf("%s mime = %s, attendu image/png ou image/webp", v.Name, v.Mime)
		}
	}
}
