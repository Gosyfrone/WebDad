package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/webdad/post-service/internal/database"
	"github.com/webdad/post-service/internal/handler"
	"github.com/webdad/post-service/internal/repository"
	"github.com/webdad/post-service/internal/service"
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

	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://root:example@localhost:27017"
	}

	client, err := database.ConnectMongo(mongoURI)
	if err != nil {
		log.Fatal(err)
	}

	db := client.Database("post_service")

	postRepo := repository.NewPostRepository(db)
	postService := service.NewPostService(postRepo)

	r := gin.Default()

	handler.RegisterRoutes(r, serviceName, postService)

	log.Printf("[%s] running on :%s", serviceName, port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}