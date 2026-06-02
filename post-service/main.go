package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/webdad/post-service/internal/handler"
)

const (
	serviceName = "post-service"
	defaultPort = "8084"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	r := gin.Default()
	handler.RegisterRoutes(r, serviceName)

	log.Printf("[%s] en écoute sur le port %s", serviceName, port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("[%s] échec du démarrage : %v", serviceName, err)
	}
}