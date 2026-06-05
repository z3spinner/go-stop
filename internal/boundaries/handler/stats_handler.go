// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/z3spinner/go-stop/internal/usecase"
)

type StatsHandler struct {
	getStats *usecase.GetStats
}

func NewStatsHandler(getStats *usecase.GetStats) *StatsHandler {
	return &StatsHandler{getStats: getStats}
}

// Get returns aggregate usage statistics.
// @ID       getStats
// @Tags     stats
// @Produce  json
// @Success  200  {object}  domain.Stats
// @Failure  500  {object}  handler.ErrorResponse
// @Router   /stats [get]
func (h *StatsHandler) Get(c *gin.Context) {
	stats, err := h.getStats.Execute()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}
