// Package imageproc génère les dérivés d'images servis par le media-service.
// L'original reste conservé ; les variantes optimisent l'affichage courant.
package imageproc

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Variant décrit un rendu compressé/redimensionné d'une image originale.
type Variant struct {
	Name   string
	Bytes  []byte
	Mime   string
	Width  int
	Height int
}

// Info résume les dimensions de l'image originale décodée.
type Info struct {
	Width  int
	Height int
}

type preset struct {
	name    string
	maxSide int
	quality int
}

var presets = []preset{
	{name: "thumb", maxSide: 320, quality: 80},
	{name: "small", maxSide: 640, quality: 82},
	{name: "medium", maxSide: 1280, quality: 84},
	{name: "large", maxSide: 2048, quality: 86},
}

// GenerateVariants produit des variantes adaptées aux usages sociaux courants.
// Sharp/libvips est utilisé en priorité pour générer du WebP ; un fallback Go
// JPEG/PNG garde le service fonctionnel dans les environnements sans Node.
func GenerateVariants(data []byte, mime string) (Info, []Variant, error) {
	if info, variants, err := generateWithSharp(data); err == nil {
		return info, variants, nil
	}
	return generateWithStdlib(data, mime)
}

func generateWithStdlib(data []byte, mime string) (Info, []Variant, error) {
	img, err := decode(data, mime)
	if err != nil {
		return Info{}, nil, err
	}
	bounds := img.Bounds()
	info := Info{Width: bounds.Dx(), Height: bounds.Dy()}
	if info.Width <= 0 || info.Height <= 0 {
		return Info{}, nil, image.ErrFormat
	}

	keepAlpha := hasAlpha(img)
	var variants []Variant
	seen := map[string]bool{}
	for _, p := range presets {
		w, h := fitInside(info.Width, info.Height, p.maxSide)
		key := sizeKey(w, h)
		if seen[key] {
			continue
		}
		seen[key] = true

		resized := resizeBilinear(img, w, h)
		encoded, outMime, err := encode(resized, keepAlpha, p.quality)
		if err != nil {
			return info, variants, err
		}
		variants = append(variants, Variant{
			Name:   p.name,
			Bytes:  encoded,
			Mime:   outMime,
			Width:  w,
			Height: h,
		})
	}
	return info, variants, nil
}

type sharpManifest struct {
	Width    int `json:"width"`
	Height   int `json:"height"`
	Variants []struct {
		Name   string `json:"name"`
		Path   string `json:"path"`
		Mime   string `json:"mime"`
		Size   int64  `json:"size"`
		Width  int    `json:"width"`
		Height int    `json:"height"`
	} `json:"variants"`
}

func generateWithSharp(data []byte) (Info, []Variant, error) {
	script, err := findSharpScript()
	if err != nil {
		return Info{}, nil, err
	}

	workDir, err := os.MkdirTemp("", "webdad-image-*")
	if err != nil {
		return Info{}, nil, err
	}
	defer func() { _ = os.RemoveAll(workDir) }()

	inputPath := filepath.Join(workDir, "input")
	outputDir := filepath.Join(workDir, "out")
	if err := os.WriteFile(inputPath, data, 0o600); err != nil {
		return Info{}, nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "node", script, inputPath, outputDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		return Info{}, nil, errorsWithOutput(err, out)
	}

	manifestBytes, err := os.ReadFile(filepath.Join(outputDir, "manifest.json"))
	if err != nil {
		return Info{}, nil, err
	}
	var manifest sharpManifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return Info{}, nil, err
	}

	var variants []Variant
	for _, v := range manifest.Variants {
		b, err := os.ReadFile(v.Path)
		if err != nil {
			return Info{}, nil, err
		}
		variants = append(variants, Variant{
			Name:   v.Name,
			Bytes:  b,
			Mime:   v.Mime,
			Width:  v.Width,
			Height: v.Height,
		})
	}
	return Info{Width: manifest.Width, Height: manifest.Height}, variants, nil
}

func findSharpScript() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		candidate := filepath.Join(dir, "scripts", "image-variants.mjs")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}

func errorsWithOutput(err error, out []byte) error {
	if len(out) == 0 {
		return err
	}
	return &sharpError{err: err, output: strings.TrimSpace(string(out))}
}

type sharpError struct {
	err    error
	output string
}

func (e *sharpError) Error() string {
	return e.err.Error() + ": " + e.output
}

func (e *sharpError) Unwrap() error {
	return e.err
}

func decode(data []byte, mime string) (image.Image, error) {
	r := bytes.NewReader(data)
	switch strings.ToLower(mime) {
	case "image/jpeg":
		return jpeg.Decode(r)
	case "image/png":
		return png.Decode(r)
	case "image/gif":
		return gif.Decode(r)
	default:
		img, _, err := image.Decode(r)
		return img, err
	}
}

func encode(img image.Image, keepAlpha bool, quality int) ([]byte, string, error) {
	var buf bytes.Buffer
	if keepAlpha {
		if err := png.Encode(&buf, img); err != nil {
			return nil, "", err
		}
		return buf.Bytes(), "image/png", nil
	}
	if err := jpeg.Encode(&buf, flatten(img), &jpeg.Options{Quality: quality}); err != nil {
		return nil, "", err
	}
	return buf.Bytes(), "image/jpeg", nil
}

func fitInside(width, height, maxSide int) (int, int) {
	longest := width
	if height > longest {
		longest = height
	}
	if longest <= maxSide {
		return width, height
	}
	scale := float64(maxSide) / float64(longest)
	return max(1, int(math.Round(float64(width)*scale))), max(1, int(math.Round(float64(height)*scale)))
}

func resizeBilinear(src image.Image, width, height int) *image.RGBA {
	srcBounds := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	if width == srcBounds.Dx() && height == srcBounds.Dy() {
		draw.Draw(dst, dst.Bounds(), src, srcBounds.Min, draw.Src)
		return dst
	}

	xRatio := float64(srcBounds.Dx()) / float64(width)
	yRatio := float64(srcBounds.Dy()) / float64(height)
	for y := 0; y < height; y++ {
		sy := (float64(y)+0.5)*yRatio - 0.5
		y0 := clampInt(int(math.Floor(sy)), 0, srcBounds.Dy()-1)
		y1 := clampInt(y0+1, 0, srcBounds.Dy()-1)
		wy := sy - math.Floor(sy)
		for x := 0; x < width; x++ {
			sx := (float64(x)+0.5)*xRatio - 0.5
			x0 := clampInt(int(math.Floor(sx)), 0, srcBounds.Dx()-1)
			x1 := clampInt(x0+1, 0, srcBounds.Dx()-1)
			wx := sx - math.Floor(sx)
			c00 := color.RGBAModel.Convert(src.At(srcBounds.Min.X+x0, srcBounds.Min.Y+y0)).(color.RGBA)
			c10 := color.RGBAModel.Convert(src.At(srcBounds.Min.X+x1, srcBounds.Min.Y+y0)).(color.RGBA)
			c01 := color.RGBAModel.Convert(src.At(srcBounds.Min.X+x0, srcBounds.Min.Y+y1)).(color.RGBA)
			c11 := color.RGBAModel.Convert(src.At(srcBounds.Min.X+x1, srcBounds.Min.Y+y1)).(color.RGBA)
			dst.SetRGBA(x, y, blend(c00, c10, c01, c11, wx, wy))
		}
	}
	return dst
}

func blend(c00, c10, c01, c11 color.RGBA, wx, wy float64) color.RGBA {
	return color.RGBA{
		R: interp(c00.R, c10.R, c01.R, c11.R, wx, wy),
		G: interp(c00.G, c10.G, c01.G, c11.G, wx, wy),
		B: interp(c00.B, c10.B, c01.B, c11.B, wx, wy),
		A: interp(c00.A, c10.A, c01.A, c11.A, wx, wy),
	}
}

func interp(v00, v10, v01, v11 uint8, wx, wy float64) uint8 {
	top := float64(v00)*(1-wx) + float64(v10)*wx
	bottom := float64(v01)*(1-wx) + float64(v11)*wx
	return uint8(math.Round(top*(1-wy) + bottom*wy))
}

func hasAlpha(img image.Image) bool {
	switch img.(type) {
	case *image.NRGBA, *image.NRGBA64, *image.RGBA, *image.RGBA64, *image.Alpha, *image.Alpha16:
		bounds := img.Bounds()
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				_, _, _, a := img.At(x, y).RGBA()
				if a != 0xffff {
					return true
				}
			}
		}
	}
	return false
}

func flatten(img image.Image) image.Image {
	bounds := img.Bounds()
	dst := image.NewRGBA(bounds)
	draw.Draw(dst, bounds, &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	draw.Draw(dst, bounds, img, bounds.Min, draw.Over)
	return dst
}

func sizeKey(width, height int) string {
	return strconv.Itoa(width) + "x" + strconv.Itoa(height)
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
