// Package models contient les structures de données du mail-service.
package models

// SendRequest : payload de POST /internal/send. Au moins l'un de HTML/Text
// doit être fourni (vérifié dans le handler).
type SendRequest struct {
	To      string `json:"to" binding:"required,email"`
	Subject string `json:"subject" binding:"required"`
	HTML    string `json:"html"`
	Text    string `json:"text"`
}
