package main

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGinMode(t *testing.T) {
	tests := map[string]string{
		gin.ReleaseMode: gin.ReleaseMode,
		gin.TestMode:    gin.TestMode,
		gin.DebugMode:   gin.DebugMode,
		"":              gin.DebugMode,
		"invalid":       gin.DebugMode,
	}
	for input, want := range tests {
		if got := ginMode(input); got != want {
			t.Errorf("ginMode(%q) = %q, attendu %q", input, got, want)
		}
	}
}
