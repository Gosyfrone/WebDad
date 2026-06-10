// Package monitoring expose un tableau de bord de santé des microservices,
// servi par le gateway (le seul à connaître l'URL de tous les services). Il
// sonde le /health de chaque service en parallèle (statut, latence, uptime).
//
// Choix : agrégation des /health plutôt que lecture du socket Docker —
// portable (marche aussi hors conteneur), sûr (pas de montage de
// `/var/run/docker.sock` ≈ root), et suffisant pour « services up + uptime ».
// Les métriques conteneur (CPU/mém/restarts via l'API Docker) sont une
// évolution possible.
package monitoring

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// startedAt : init au démarrage du process gateway → uptime du gateway lui-même.
var startedAt = time.Now()

// ServiceHealth : état d'un composant à l'instant de la sonde.
type ServiceHealth struct {
	Name          string `json:"name"`
	Prefix        string `json:"prefix,omitempty"`
	Status        string `json:"status"` // "up" | "down"
	LatencyMs     int64  `json:"latency_ms"`
	UptimeSeconds int    `json:"uptime_seconds"`
	Error         string `json:"error,omitempty"`
}

// upstreamHealth : forme attendue d'une réponse /health (les services exposent
// désormais `uptime_seconds`).
type upstreamHealth struct {
	Status        string `json:"status"`
	Service       string `json:"service"`
	UptimeSeconds int    `json:"uptime_seconds"`
}

// Handler : GET /admin/monitoring — agrège la santé du gateway + de tous les
// services. À protéger par une garde admin (cf. router).
func Handler(services map[string]string) gin.HandlerFunc {
	client := &http.Client{Timeout: 3 * time.Second}
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"data": gin.H{
			"generated_at": time.Now().UTC(),
			"services":     probeAll(c.Request.Context(), client, services),
		}})
	}
}

// probeAll sonde tous les services en parallèle et préfixe le résultat par le
// gateway lui-même (toujours « up » puisqu'il répond à cette requête).
func probeAll(ctx context.Context, client *http.Client, services map[string]string) []ServiceHealth {
	out := []ServiceHealth{{
		Name:          "api-gateway",
		Status:        "up",
		LatencyMs:     0,
		UptimeSeconds: int(time.Since(startedAt).Seconds()),
	}}

	prefixes := make([]string, 0, len(services))
	for p := range services {
		prefixes = append(prefixes, p)
	}
	sort.Strings(prefixes) // ordre déterministe

	probed := make([]ServiceHealth, len(prefixes))
	done := make(chan int, len(prefixes))
	for i, prefix := range prefixes {
		go func(i int, prefix, target string) {
			probed[i] = probeOne(ctx, client, prefix, target)
			done <- i
		}(i, prefix, services[prefix])
	}
	for range prefixes {
		<-done
	}
	return append(out, probed...)
}

// probeOne effectue un GET <target>/health, mesure la latence et lit l'uptime.
func probeOne(ctx context.Context, client *http.Client, prefix, target string) ServiceHealth {
	h := ServiceHealth{Name: strings.TrimPrefix(prefix, "/"), Prefix: prefix, Status: "down"}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target+"/health", nil)
	if err != nil {
		h.Error = err.Error()
		return h
	}
	start := time.Now()
	resp, err := client.Do(req)
	h.LatencyMs = time.Since(start).Milliseconds()
	if err != nil {
		h.Error = "injoignable"
		return h
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		h.Error = resp.Status
		return h
	}
	h.Status = "up"
	var body upstreamHealth
	if json.NewDecoder(resp.Body).Decode(&body) == nil {
		h.UptimeSeconds = body.UptimeSeconds
		if body.Service != "" {
			h.Name = body.Service
		}
	}
	return h
}
