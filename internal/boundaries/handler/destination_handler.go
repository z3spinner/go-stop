// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/z3spinner/go-stop/internal/usecase"
)

type DestinationHandler struct {
	getDestinations *usecase.GetDestinations
}

func NewDestinationHandler(getDestinations *usecase.GetDestinations) *DestinationHandler {
	return &DestinationHandler{getDestinations: getDestinations}
}

// List returns all known destinations (for autocomplete).
// @ID       listDestinations
// @Tags     destinations
// @Produce  json
// @Success  200  {array}  string
// @Failure  500  {object}  handler.ErrorResponse
// @Router   /destinations [get]
func (h *DestinationHandler) List(c *gin.Context) {
	destinations, err := h.getDestinations.Execute()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, destinations)
}
