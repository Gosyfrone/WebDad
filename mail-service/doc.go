package main

// @title           Breezy — Mail Service
// @version         1.0
// @description     Transport e-mail (SMTP Gmail) pour Breezy. Endpoint interne
// @description     serveur-à-serveur (POST /internal/send), protégé par un secret
// @description     partagé et jamais routé par l'API Gateway ni exposé au client.
// @host            localhost:8089
// @BasePath        /
