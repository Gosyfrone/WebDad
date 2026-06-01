package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
)

const (
	serviceName = "auth-service"
	defaultPort = "3001"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": serviceName,
		})
	})

	log.Printf("[%s] en écoute sur le port %s", serviceName, port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("[%s] échec du démarrage : %v", serviceName, err)
	}
}
