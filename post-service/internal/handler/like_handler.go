package handler

import (
    "github.com/gin-gonic/gin"
)

func ListPostLikes(service string) gin.HandlerFunc {
	//TODO Replace
    return func(c *gin.Context) {
        c.JSON(200, gin.H{
            "status":  "ok",
            "service": service,
        })
    }
}

func LikePost(service string) gin.HandlerFunc {
	//TODO Replace
    return func(c *gin.Context) {
        c.JSON(200, gin.H{
            "status":  "ok",
            "service": service,
        })
    }
}


func UnlikePost(service string) gin.HandlerFunc {
	//TODO Replace
    return func(c *gin.Context) {
        c.JSON(200, gin.H{
            "status":  "ok",
            "service": service,
        })
    }
}


func ListProfileLikes(service string) gin.HandlerFunc {
	//TODO Replace
    return func(c *gin.Context) {
        c.JSON(200, gin.H{
            "status":  "ok",
            "service": service,
        })
    }
}