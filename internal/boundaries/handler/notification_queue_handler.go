package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/z3spinner/go-stop/internal/usecase"
)

type NotificationQueueHandler struct {
	getPending *usecase.GetPendingNotifications
}

func NewNotificationQueueHandler(getPending *usecase.GetPendingNotifications) *NotificationQueueHandler {
	return &NotificationQueueHandler{getPending: getPending}
}

// List returns pending ride notifications for the authenticated searcher.
// GET /api/notifications?phone=... or X-Phone header.
func (h *NotificationQueueHandler) List(c *gin.Context) {
	phone := normalizePhone(c.GetHeader("X-Phone"))
	if phone == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "X-Phone header required"})
		return
	}
	summaries, err := h.getPending.Execute(phone)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	type item struct {
		RideID      string `json:"ride_id"`
		RequestID   string `json:"request_id"`
		DriverName  string `json:"driver_name"`
		Origin      string `json:"origin"`
		Destination string `json:"destination"`
		DepartureAt string `json:"departure_at"`
		SentCount   int    `json:"sent_count"`
	}
	out := make([]item, len(summaries))
	for i, s := range summaries {
		out[i] = item{
			RideID:      s.Entry.RideID,
			RequestID:   s.Entry.RequestID,
			DriverName:  s.Ride.DriverName,
			Origin:      s.Ride.Origin,
			Destination: s.Ride.Destination,
			DepartureAt: s.Ride.DepartureAt.Format("2006-01-02T15:04:05Z"),
			SentCount:   s.Entry.SentCount,
		}
	}
	c.JSON(http.StatusOK, out)
}
