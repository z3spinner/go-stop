package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type ConfigHandler struct {
	siteName string
}

func NewConfigHandler(siteName string) *ConfigHandler {
	return &ConfigHandler{siteName: siteName}
}

func (h *ConfigHandler) Get(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"siteName": h.siteName})
}
