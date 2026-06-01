package handler

import (
    "github.com/gin-gonic/gin"
)

func Health(service string) gin.HandlerFunc {
    return func(c *gin.Context) {
        c.JSON(200, gin.H{
            "status":  "ok",
            "service": service,
        })
    }
}