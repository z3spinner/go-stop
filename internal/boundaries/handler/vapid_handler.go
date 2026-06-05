// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

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

// GetPublicKey returns the server's VAPID public key for Web Push.
// @ID       getVapidPublicKey
// @Tags     vapid
// @Produce  json
// @Success  200  {object}  handler.VapidKeyResponse
// @Router   /vapid-public-key [get]
func (h *VapidHandler) GetPublicKey(c *gin.Context) {
	c.JSON(http.StatusOK, VapidKeyResponse{PublicKey: h.publicKey})
}
