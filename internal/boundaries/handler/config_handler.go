// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type ConfigHandler struct {
	siteName         string
	returnDelayHours int
}

func NewConfigHandler(siteName string, returnDelayHours int) *ConfigHandler {
	return &ConfigHandler{siteName: siteName, returnDelayHours: returnDelayHours}
}

// Get returns the public site configuration.
// @ID       getConfig
// @Tags     config
// @Produce  json
// @Success  200  {object}  handler.ConfigResponse
// @Router   /config [get]
func (h *ConfigHandler) Get(c *gin.Context) {
	c.JSON(http.StatusOK, ConfigResponse{
		SiteName:         h.siteName,
		ReturnDelayHours: h.returnDelayHours,
	})
}
