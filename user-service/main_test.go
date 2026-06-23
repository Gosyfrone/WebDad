package main

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGinMode(t *testing.T) {
	cases := map[string]string{
		gin.ReleaseMode: gin.ReleaseMode,
		gin.TestMode:    gin.TestMode,
		gin.DebugMode:   gin.DebugMode,
		"":              gin.DebugMode,
		"inconnu":       gin.DebugMode,
	}
	for in, want := range cases {
		if got := ginMode(in); got != want {
			t.Errorf("ginMode(%q) = %q, attendu %q", in, got, want)
		}
	}
}
