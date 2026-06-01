package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type ConfigHandler struct {
	siteName          string
	returnDelayHours  int
}

func NewConfigHandler(siteName string, returnDelayHours int) *ConfigHandler {
	return &ConfigHandler{siteName: siteName, returnDelayHours: returnDelayHours}
}

func (h *ConfigHandler) Get(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"siteName":         h.siteName,
		"returnDelayHours": h.returnDelayHours,
	})
}
