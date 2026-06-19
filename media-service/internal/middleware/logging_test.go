package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// ─── RequestID tests ─────────────────────────────────────────────────────────

func TestRequestID_GeneréSiAbsent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/test", RequestID(), func(c *gin.Context) {
		id := c.GetString("request_id")
		if id == "" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "request_id vide"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"request_id": id})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, attendu 200 (corps: %s)", w.Code, w.Body.String())
	}
	if w.Header().Get("X-Request-Id") == "" {
		t.Fatal("en-tête X-Request-Id absent de la réponse")
	}
}

func TestRequestID_PropagéSiPrésent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/test", RequestID(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"request_id": c.GetString("request_id")})
	})

	const wantID = "mon-request-id-custom"
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Request-Id", wantID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, attendu 200", w.Code)
	}
	got := w.Header().Get("X-Request-Id")
	if got != wantID {
		t.Fatalf("X-Request-Id = %q, attendu %q", got, wantID)
	}
}

func TestRequestID_FormatHex(t *testing.T) {
	// Sans X-Request-Id entrant, l'id généré doit être un hex de 32 caractères.
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/test", RequestID(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	id := w.Header().Get("X-Request-Id")
	if len(id) != 32 {
		t.Fatalf("longueur X-Request-Id = %d, attendu 32 (hex 16 octets)", len(id))
	}
	for _, c := range id {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			t.Fatalf("X-Request-Id non-hex : %q", id)
		}
	}
}

func TestRequestID_UniqueParRequête(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/test", RequestID(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{})
	})

	ids := make(map[string]bool)
	for i := 0; i < 20; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		id := w.Header().Get("X-Request-Id")
		if ids[id] {
			t.Fatalf("collision d'id après %d requêtes : %s", i, id)
		}
		ids[id] = true
	}
}

// ─── RequestLogger tests ─────────────────────────────────────────────────────

func TestRequestLogger_2xx(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/test", RequestLogger(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	// Ne doit pas paniquer.
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, attendu 200", w.Code)
	}
}

func TestRequestLogger_4xx(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/test", RequestLogger(), func(c *gin.Context) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, attendu 400", w.Code)
	}
}

func TestRequestLogger_5xx(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/test", RequestLogger(), func(c *gin.Context) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, attendu 500", w.Code)
	}
}

func TestRequestLogger_AvecUserID(t *testing.T) {
	// Vérifie que le logger ne panique pas quand user_id est présent.
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		c.Set("user_id", "42")
		c.Next()
	}, RequestLogger(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, attendu 200", w.Code)
	}
}

func TestRequestLogger_AvecRequestID(t *testing.T) {
	// RequestID + RequestLogger ensemble.
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/test", RequestID(), RequestLogger(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, attendu 200", w.Code)
	}
	if w.Header().Get("X-Request-Id") == "" {
		t.Fatal("X-Request-Id absent")
	}
}

// ─── Recovery tests ──────────────────────────────────────────────────────────

func TestRecovery_PanicCapturée(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/test", Recovery(), func(c *gin.Context) {
		panic("erreur inattendue")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("panic non capturée : status = %d, attendu 500", w.Code)
	}
}

func TestRecovery_PanicAvecRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/test", RequestID(), Recovery(), func(c *gin.Context) {
		panic("boom")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("attendu 500 après panic, obtenu %d", w.Code)
	}
}

func TestRecovery_SansPanic_PasseThrough(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/test", Recovery(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("sans panic : attendu 200, obtenu %d", w.Code)
	}
}

// ─── AdminOnly sans claims (branche "non authentifié" d'AdminOnly) ──────────

func TestAdminOnly_SansClaims(t *testing.T) {
	// Cas où AdminOnly est appelé sans JWTAuth en amont (pas de claims dans le contexte).
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/admin", AdminOnly(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("AdminOnly sans claims : attendu 401, obtenu %d", w.Code)
	}
}
