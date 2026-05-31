package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type VapidHandler struct {
	publicKey string
}

func NewVapidHandler(publicKey string) *VapidHandler {
	return &VapidHandler{publicKey: publicKey}
}

func (h *VapidHandler) GetPublicKey(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"publicKey": h.publicKey})
}
