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

	rideH := handler.NewRideHandler(postRide, getRides, getMyRides, searchRides, deleteRide, getMatchingRequests, statRepo, interestRepo, rideRepo)
	interestH := handler.NewInterestHandler(expressInterest, acceptInterest, getInterestContact)
	requestH := handler.NewRequestHandler(postRequest, getMyRequests, deleteRequest, requestRepo)
	destH := handler.NewDestinationHandler(getDests)
	subH := handler.NewSubscriptionHandler(subscribe, unsubscribe)
	vapidH := handler.NewVapidHandler(os.Getenv("VAPID_PUBLIC_KEY"))
	siteName := os.Getenv("SITE_NAME")
	if siteName == "" {
		siteName = "Go-Stop"
	}
	configH := handler.NewConfigHandler(siteName)
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
	r.SetTrustedProxies(nil)
	r.Static("/css", "./web/css")
	r.Static("/js", "./web/js")
	r.StaticFile("/manifest.json", "./web/manifest.json")
	r.StaticFile("/sw.js", "./web/js/sw.js")
	r.StaticFile("/logo.svg", "./web/logo.svg")
	r.NoRoute(func(c *gin.Context) {
		c.File("./web/index.html")
	})

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
		api.GET("/interests/:id/contact", interestH.GetContact)

		api.POST("/requests", requestH.Post)
		api.GET("/requests", requestH.List)
		api.GET("/requests/:id", requestH.Get)
		api.DELETE("/requests/:id", requestH.Delete)

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
