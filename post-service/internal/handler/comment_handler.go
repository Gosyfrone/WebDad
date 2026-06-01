package handler

import (
    "github.com/gin-gonic/gin"
)

func ListPostComments(service string) gin.HandlerFunc {
	//TODO Replace
    return func(c *gin.Context) {
        c.JSON(200, gin.H{
            "status":  "ok",
            "service": service,
        })
    }
}

func CreatPostComment(service string) gin.HandlerFunc {
	//TODO Replace
    return func(c *gin.Context) {
        c.JSON(200, gin.H{
            "status":  "ok",
            "service": service,
        })
    }
}

func DeletePostComment(service string) gin.HandlerFunc {
	//TODO Replace
    return func(c *gin.Context) {
        c.JSON(200, gin.H{
            "status":  "ok",
            "service": service,
        })
    }
}

func ListProfileComments(service string) gin.HandlerFunc {
	//TODO Replace
    return func(c *gin.Context) {
        c.JSON(200, gin.H{
            "status":  "ok",
            "service": service,
        })
    }
}

func CreateComment(service string) gin.HandlerFunc {
	//TODO Replace
    return func(c *gin.Context) {
        c.JSON(200, gin.H{
            "status":  "ok",
            "service": service,
        })
    }
}