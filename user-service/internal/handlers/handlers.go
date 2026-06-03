// Package handlers expose les handlers HTTP (Gin) du service user.
// Réponses standardisées : succès = {"data": ...}, erreur = {"error": ...}.
package handlers

import "github.com/webdad/user-service/internal/service"

// Handler porte les dépendances partagées par les handlers.
type Handler struct {
	users *service.UserService
}

// New construit le groupe de handlers.
func New(users *service.UserService) *Handler {
	return &Handler{users: users}
}
