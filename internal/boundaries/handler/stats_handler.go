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

func (h *StatsHandler) Get(c *gin.Context) {
	stats, err := h.getStats.Execute()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}
