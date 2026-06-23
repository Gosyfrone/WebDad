package main

import (
	"errors"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/webdad/media-service/internal/config"
	"github.com/webdad/media-service/internal/storage"
)

func TestGinMode(t *testing.T) {
	if ginMode(gin.ReleaseMode) != gin.ReleaseMode || ginMode(gin.TestMode) != gin.TestMode || ginMode("invalid") != gin.DebugMode {
		t.Fatal("mode")
	}
}
func TestRun(t *testing.T) {
	cfg := &config.Config{Port: "1234", GinMode: "test"}
	boom := errors.New("boom")
	badFactory := func(string, string, string, bool, string) (*storage.Store, error) { return nil, boom }
	if err := run(cfg, badFactory, func(*gin.Engine, string) error { return nil }); !errors.Is(err, boom) {
		t.Fatal(err)
	}
	goodFactory := func(string, string, string, bool, string) (*storage.Store, error) { return nil, nil }
	if err := run(cfg, goodFactory, func(r *gin.Engine, address string) error {
		if address != ":1234" || len(r.Routes()) == 0 {
			t.Fatal(address)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := run(cfg, goodFactory, func(*gin.Engine, string) error { return boom }); !errors.Is(err, boom) {
		t.Fatal(err)
	}
}
