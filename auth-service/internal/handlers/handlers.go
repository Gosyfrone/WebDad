// Package handlers expose les handlers HTTP (Gin) du service auth.
// Réponses standardisées : succès = {"data": ...}, erreur = {"error": ...}.
package handlers

import "github.com/webdad/auth-service/internal/services"

// Handler porte les dépendances partagées par les handlers.
type Handler struct {
	auth *services.AuthService
}

// New construit le groupe de handlers.
func New(auth *services.AuthService) *Handler {
	return &Handler{auth: auth}
}
