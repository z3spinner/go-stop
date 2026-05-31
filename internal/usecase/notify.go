package usecase

import (
	"fmt"

	"github.com/z3spinner/go-stop/internal/boundaries/notification"
	"github.com/z3spinner/go-stop/internal/domain"
)

func NotifySearcher(sub domain.Subscription, ride domain.Ride, notifier notification.Notifier) error {
	msg := domain.Message{
		Title:       "Ride available!",
		Body:        fmt.Sprintf("%s is driving from %s to %s", ride.DriverName, ride.Origin, ride.Destination),
		URL:         "/rides/" + ride.ID,
		ContactName: ride.DriverName,
		Phone:       ride.Phone,
		Origin:      ride.Origin,
		Destination: ride.Destination,
		DepartureAt: ride.DepartureAt,
	}
	return notifier.Send(sub, msg)
}

func NotifyDriver(sub domain.Subscription, req domain.Request, notifier notification.Notifier) error {
	msg := domain.Message{
		Title:       "Someone needs a ride!",
		Body:        fmt.Sprintf("%s needs a ride from %s to %s", req.SearcherName, req.Origin, req.Destination),
		URL:         "/requests/" + req.ID,
		ContactName: req.SearcherName,
		Phone:       req.Phone,
		Origin:      req.Origin,
		Destination: req.Destination,
		DepartureAt: req.DepartureAt,
	}
	return notifier.Send(sub, msg)
}
