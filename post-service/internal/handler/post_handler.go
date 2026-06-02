package handler

import (
    "github.com/gin-gonic/gin"
)

func ListPosts(service string) gin.HandlerFunc {
	//TODO Replace
    return func(c *gin.Context) {
        c.JSON(200, gin.H{
            "status":  "ok",
            "service": service,
        })
    }
}

func CreatePost(service string) gin.HandlerFunc {
	//TODO Replace
    return func(c *gin.Context) {
        c.JSON(200, gin.H{
            "status":  "ok",
            "service": service,
        })
    }
}

func GetPost(service string) gin.HandlerFunc {
	//TODO Replace
    return func(c *gin.Context) {
        c.JSON(200, gin.H{
            "status":  "ok",
            "service": service,
        })
    }
}

func DeletePost(service string) gin.HandlerFunc {
	//TODO Replace
    return func(c *gin.Context) {
        c.JSON(200, gin.H{
            "status":  "ok",
            "service": service,
        })
    }
}

func ListProfilePosts(service string) gin.HandlerFunc {
	//TODO Replace
    return func(c *gin.Context) {
        c.JSON(200, gin.H{
            "status":  "ok",
            "service": service,
        })
    }
}