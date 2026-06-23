package config

import "testing"

func TestLoadAndEnvironmentHelpers(t *testing.T) {
	t.Setenv("JWT_SECRET", "jwt")
	t.Setenv("MINIO_ROOT_USER", "user")
	t.Setenv("MINIO_ROOT_PASSWORD", "pass")
	t.Setenv("PORT", "9999")
	t.Setenv("GIN_MODE", "release")
	t.Setenv("MINIO_ENDPOINT", "minio:9000")
	t.Setenv("MINIO_USE_SSL", "true")
	t.Setenv("MINIO_BUCKET", "bucket")
	t.Setenv("MEDIA_MAX_IMAGE_BYTES", "11")
	t.Setenv("MEDIA_MAX_VIDEO_BYTES", "bad")
	t.Setenv("MEDIA_MAX_BLOB_BYTES", "13")
	t.Setenv("GIPHY_API_KEY", "giphy")
	cfg := Load()
	if cfg.Port != "9999" || cfg.JWTSecret != "jwt" || !cfg.MinioUseSSL || cfg.MaxImageBytes != 11 || cfg.MaxVideoBytes != defaultMaxVideoBytes || cfg.MaxBlobBytes != 13 || cfg.GiphyAPIKey != "giphy" {
		t.Fatalf("config inattendue: %#v", cfg)
	}
	t.Setenv("EMPTY_VALUE", "")
	if getEnv("EMPTY_VALUE", "fallback") != "fallback" || getInt64("EMPTY_VALUE", 7) != 7 || !getBool("EMPTY_VALUE", true) {
		t.Fatal("fallbacks")
	}
	t.Setenv("BOOL_BAD", "certainement")
	if getBool("BOOL_BAD", true) != true {
		t.Fatal("bool invalide")
	}
}
