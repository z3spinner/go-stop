package usecase

import (
	"github.com/z3spinner/go-stop/internal/boundaries/repository"
	"github.com/z3spinner/go-stop/internal/domain"
)

type GetStats struct {
	stats repository.StatRepository
}

func NewGetStats(stats repository.StatRepository) *GetStats {
	return &GetStats{stats: stats}
}

func (uc *GetStats) Execute() (domain.Stats, error) {
	return uc.stats.GetStats()
}
