package main

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/z3spinner/go-stop/internal/boundaries/handler"
	"github.com/z3spinner/go-stop/internal/infrastructure/postgres"
	"github.com/z3spinner/go-stop/internal/infrastructure/webpush"
	"github.com/z3spinner/go-stop/internal/usecase"
	"github.com/z3spinner/go-stop/internal/version"
)

func main() {
	pool, err := postgres.NewPool()
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()

	graceMins := 60
	if v := os.Getenv("RIDE_GRACE_MINUTES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			graceMins = n
		}
	}
	rideRepo := postgres.NewRideRepo(pool, graceMins)
	requestRepo := postgres.NewRequestRepo(pool)
	destRepo := postgres.NewDestinationRepo(pool)
	subRepo := postgres.NewSubscriptionRepo(pool)
	statRepo := postgres.NewStatRepo(pool)
	interestRepo := postgres.NewInterestRepo(pool)

	notifier := webpush.New(
		os.Getenv("VAPID_PUBLIC_KEY"),
		os.Getenv("VAPID_PRIVATE_KEY"),
		os.Getenv("VAPID_EMAIL"),
	)

	postRide := usecase.NewPostRide(rideRepo, requestRepo, subRepo, notifier)
	postRequest := usecase.NewPostRequest(requestRepo, rideRepo, subRepo, notifier)
	getRides := usecase.NewGetRides(rideRepo)
	getMyRides := usecase.NewGetMyRides(rideRepo)
	searchRides := usecase.NewSearchRides(rideRepo)
	getDests := usecase.NewGetDestinations(destRepo)
	subscribe := usecase.NewSubscribe(subRepo)
	unsubscribe := usecase.NewUnsubscribe(subRepo)
	deleteRide := usecase.NewDeleteRide(rideRepo)
	deleteRequest := usecase.NewDeleteRequest(requestRepo)
	pingSearcher := usecase.NewPingSearcher(requestRepo, rideRepo, interestRepo, subRepo, notifier)
	getMyRequests := usecase.NewGetMyRequests(requestRepo)
	expireRides := usecase.NewExpireRides(rideRepo)
	expireRequests := usecase.NewExpireRequests(requestRepo)
	getMatchingRequests := usecase.NewGetMatchingRequests(rideRepo, requestRepo)
	recordFeedback := usecase.NewRecordFeedback(rideRepo, statRepo)
	getStats := usecase.NewGetStats(statRepo)
	sendFeedbackReminders := usecase.NewSendFeedbackReminders(rideRepo, subRepo, notifier)
	expressInterest    := usecase.NewExpressInterest(rideRepo, interestRepo, subRepo, notifier)
	acceptInterest     := usecase.NewAcceptInterest(interestRepo, rideRepo, subRepo, notifier)
	getInterestContact := usecase.NewGetInterestContact(interestRepo, rideRepo)

	serviceTZ := time.UTC
	if tzName := os.Getenv("SERVICE_TZ"); tzName != "" {
		if loc, err := time.LoadLocation(tzName); err == nil {
			serviceTZ = loc
			log.Printf("service timezone: %s", tzName)
		} else {
			log.Printf("warning: invalid SERVICE_TZ %q, using UTC: %v", tzName, err)
		}
	}

	rideH := handler.NewRideHandler(postRide, getRides, getMyRides, searchRides, deleteRide, getMatchingRequests, statRepo, interestRepo, rideRepo, serviceTZ)
	interestH := handler.NewInterestHandler(expressInterest, acceptInterest, getInterestContact, interestRepo)
	requestH := handler.NewRequestHandler(postRequest, getMyRequests, deleteRequest, pingSearcher, requestRepo)
	destH := handler.NewDestinationHandler(getDests)
	subH := handler.NewSubscriptionHandler(subscribe, unsubscribe)
	vapidH := handler.NewVapidHandler(os.Getenv("VAPID_PUBLIC_KEY"))

	siteName := os.Getenv("SITE_NAME")
	if siteName == "" {
		siteName = "Go-Stop"
	}
	returnDelayHours := 2
	if v := os.Getenv("RETURN_DELAY_HOURS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			returnDelayHours = n
		}
	}
	configH := handler.NewConfigHandler(siteName, returnDelayHours)
	feedbackH := handler.NewFeedbackHandler(recordFeedback)
	statsH := handler.NewStatsHandler(getStats)

	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			if err := expireRides.Execute(); err != nil {
				log.Printf("expire rides: %v", err)
			}
			if err := expireRequests.Execute(); err != nil {
				log.Printf("expire requests: %v", err)
			}
			if err := sendFeedbackReminders.Execute(); err != nil {
				log.Printf("send feedback reminders: %v", err)
			}
		}
	}()

	r := gin.Default()
	// On Scalingo the real client IP is in X-Real-IP (set by their reverse proxy).
	r.TrustedPlatform = "X-Real-IP"
	r.Static("/css", "./web/css")
	r.Static("/js", "./web/js")
	r.StaticFile("/manifest.json", "./web/manifest.json")
	r.StaticFile("/sw.js", "./web/js/sw.js")
	r.StaticFile("/logo.svg", "./web/logo.svg")
	buildVersion := version.Get()
	indexH, err := handler.NewIndexHandler("./web/index.html", buildVersion)
	if err != nil {
		log.Fatalf("index template: %v", err)
	}
	log.Printf("build version: %s", buildVersion)
	r.NoRoute(indexH.Serve)

	api := r.Group("/api")
	{
		api.POST("/rides", rideH.Post)
		api.GET("/rides", rideH.List)
		api.GET("/rides/:id", rideH.Get)
		api.DELETE("/rides/:id", rideH.Delete)
		api.GET("/rides/:id/requests", rideH.ListMatchingRequests)
		api.POST("/rides/:id/feedback", feedbackH.Post)
		api.GET("/rides/:id/interests",   rideH.ListInterests)
		api.POST("/rides/:id/interest",   interestH.Express)
		api.POST("/interests/:id/accept", interestH.Accept)
		api.GET("/interests", interestH.ListMyRequests)
		api.GET("/interests/:id/contact", interestH.GetContact)

		api.POST("/requests", requestH.Post)
		api.GET("/requests", requestH.List)
		api.GET("/requests/:id", requestH.Get)
		api.DELETE("/requests/:id", requestH.Delete)
		api.POST("/requests/:id/ping", requestH.Ping)

		api.GET("/destinations", destH.List)

		api.POST("/subscriptions", subH.Subscribe)
		api.DELETE("/subscriptions/:phone", subH.Unsubscribe)

		api.GET("/vapid-public-key", vapidH.GetPublicKey)
		api.GET("/config", configH.Get)
		api.GET("/stats", statsH.Get)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("listening on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("server: %v", err)
	}
}
