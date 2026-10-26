// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/z3spinner/go-stop/internal/boundaries/handler"
	"github.com/z3spinner/go-stop/internal/infrastructure/postgres"
	"github.com/z3spinner/go-stop/internal/infrastructure/vapid"
	"github.com/z3spinner/go-stop/internal/infrastructure/webpush"
	"github.com/z3spinner/go-stop/internal/usecase"
	"github.com/z3spinner/go-stop/internal/version"
)

// @title        Go-Stop API
// @version      1.0
// @description  Local ride-sharing notice board API.
// @BasePath     /api
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
	serviceTZ := time.UTC
	if tzName := os.Getenv("SERVICE_TZ"); tzName != "" {
		if loc, err := time.LoadLocation(tzName); err == nil {
			serviceTZ = loc
			log.Printf("service timezone: %s", tzName)
		} else {
			log.Printf("warning: invalid SERVICE_TZ %q, using UTC: %v", tzName, err)
		}
	}
	rideRepo := postgres.NewRideRepo(pool, graceMins, serviceTZ)
	requestRepo := postgres.NewRequestRepo(pool, graceMins, serviceTZ)
	destRepo := postgres.NewDestinationRepo(pool)
	subRepo := postgres.NewSubscriptionRepo(pool)
	statRepo := postgres.NewStatRepo(pool)
	interestRepo := postgres.NewInterestRepo(pool)

	vapidKeys, vapidSource, err := vapid.Resolve(context.Background(), postgres.NewSettingsRepo(pool), os.Getenv)
	if err != nil {
		log.Fatalf("vapid: %v", err)
	}
	log.Printf("vapid: keys ready (source: %s)", vapidSource)

	// Every push with a Kind is also kept in the recipient's in-app inbox, so a
	// missed notification can still be read in the app (see usecase/inbox.go).
	inboxRepo := postgres.NewInboxRepo(pool)
	notifier := usecase.NewInboxNotifier(webpush.New(vapidKeys.Public, vapidKeys.Private, vapidKeys.Email), inboxRepo)
	inboxRetentionDays := usecase.DefaultInboxRetentionDays
	if v := os.Getenv("NOTIFICATION_RETENTION_DAYS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			inboxRetentionDays = usecase.ClampInboxRetentionDays(n)
		}
	}
	getInbox := usecase.NewGetInbox(inboxRepo)
	markInboxRead := usecase.NewMarkInboxRead(inboxRepo)
	deleteInboxItem := usecase.NewDeleteInboxItem(inboxRepo)
	restoreInboxItem := usecase.NewRestoreInboxItem(inboxRepo)
	purgeInbox := usecase.NewPurgeInbox(inboxRepo, inboxRetentionDays)

	notifQueueRepo := postgres.NewNotificationQueueRepo(pool)
	feedbackQueueRepo := postgres.NewFeedbackQueueRepo(pool)
	contactOfferRepo := postgres.NewContactOfferRepo(pool)

	postRide := usecase.NewPostRide(rideRepo, requestRepo, subRepo, notifQueueRepo, notifier, graceMins)
	updateRide := usecase.NewUpdateRide(rideRepo, requestRepo, subRepo, notifQueueRepo, notifier, graceMins)
	postRequest := usecase.NewPostRequest(requestRepo, rideRepo, subRepo, notifier)
	getRides := usecase.NewGetRides(rideRepo)
	getMyRides := usecase.NewGetMyRides(rideRepo)
	searchRides := usecase.NewSearchRides(rideRepo)
	getDests := usecase.NewGetDestinations(destRepo)
	subscribe := usecase.NewSubscribe(subRepo)
	unsubscribe := usecase.NewUnsubscribe(subRepo)
	sendTestPush := usecase.NewSendTestPush(subRepo, notifier)
	deleteRide := usecase.NewDeleteRide(rideRepo, notifQueueRepo, feedbackQueueRepo, inboxRepo)
	deleteRequest := usecase.NewDeleteRequest(requestRepo, inboxRepo)
	pingSearcher := usecase.NewPingSearcher(requestRepo, rideRepo, interestRepo, subRepo, notifier)
	getMyRequests := usecase.NewGetMyRequests(requestRepo)
	getActiveRequests := usecase.NewGetActiveRequests(requestRepo)
	expireRides := usecase.NewExpireRides(rideRepo)
	expireRequests := usecase.NewExpireRequests(requestRepo)
	getMatchingRequests := usecase.NewGetMatchingRequests(rideRepo, requestRepo)
	expireMatchQueue := usecase.NewExpireMatchQueue(notifQueueRepo)
	getPendingNotifications := usecase.NewGetPendingNotifications(notifQueueRepo, rideRepo)
	recordFeedback := usecase.NewRecordFeedback(rideRepo, statRepo, feedbackQueueRepo, notifier)
	getStats := usecase.NewGetStats(statRepo)
	enqueueFeedback := usecase.NewEnqueueFeedback(feedbackQueueRepo)
	expressInterest := usecase.NewExpressInterest(rideRepo, interestRepo, subRepo, notifier)
	acceptInterest := usecase.NewAcceptInterest(interestRepo, rideRepo, subRepo, notifier)
	getInterestContact := usecase.NewGetInterestContact(interestRepo, rideRepo)
	cancelInterest := usecase.NewCancelInterest(interestRepo, rideRepo, subRepo, notifier)
	offerContact := usecase.NewOfferContact(requestRepo, contactOfferRepo, subRepo, notifier)
	getContactOfferStatus := usecase.NewGetRequestContactOfferStatus(requestRepo, contactOfferRepo)
	getRequestContactOffers := usecase.NewGetRequestContactOffers(requestRepo, contactOfferRepo)
	renewRequest := usecase.NewRenewRequest(requestRepo, requestRepo, notifier)

	settingsRepo := postgres.NewSettingsRepo(pool)
	sendEngagementReminder := usecase.NewSendEngagementReminder(rideRepo, subRepo, settingsRepo, notifier, serviceTZ)

	// Reminder pushes for unread notifications about another person's action.
	reminderHours := usecase.DefaultInboxReminderIntervalHours
	if v := os.Getenv("INBOX_REMINDER_INTERVAL_HOURS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			reminderHours = n
		}
	}
	quietSetting := usecase.DefaultInboxReminderQuietHours
	if v, ok := os.LookupEnv("INBOX_REMINDER_QUIET_HOURS"); ok {
		quietSetting = v
	}
	quiet, err := usecase.ParseQuietHours(quietSetting)
	if err != nil {
		log.Printf("warning: %v; using %s", err, usecase.DefaultInboxReminderQuietHours)
		quiet, _ = usecase.ParseQuietHours(usecase.DefaultInboxReminderQuietHours)
	}
	sendInboxReminders := usecase.NewSendInboxReminders(inboxRepo, subRepo, notifier, reminderHours, quiet, serviceTZ)
	// The post-ride feedback prompt is pushed once (not during quiet hours);
	// its repeats are inbox reminders.
	sendFeedbackPrompts := usecase.NewSendFeedbackPrompts(feedbackQueueRepo, subRepo, notifier, quiet, serviceTZ)
	// Daily summary of the day's board, on by default (per-phone off switch).
	notifPrefsRepo := postgres.NewNotificationPreferenceRepo(pool)
	digestHour, err := usecase.ParseMorningDigestHour(os.Getenv("MORNING_DIGEST_HOUR"))
	if err != nil {
		log.Printf("warning: %v; using %d", err, usecase.DefaultMorningDigestHour)
		digestHour = usecase.DefaultMorningDigestHour
	}
	// Open-ended alerts last 30 days; their owners are asked 3 days before
	// whether to keep them (not during quiet hours).
	sendAlertExpiryWarnings := usecase.NewSendAlertExpiryWarnings(requestRepo, subRepo, notifier, quiet, serviceTZ)

	rideH := handler.NewRideHandler(postRide, updateRide, getRides, getMyRides, searchRides, deleteRide, getMatchingRequests, statRepo, interestRepo, rideRepo, serviceTZ)
	interestH := handler.NewInterestHandler(expressInterest, acceptInterest, getInterestContact, cancelInterest, interestRepo, statRepo)
	reqH := handler.NewRequestHandler(postRequest, getMyRequests, getActiveRequests, deleteRequest, pingSearcher, offerContact, getContactOfferStatus, getRequestContactOffers, renewRequest, requestRepo, statRepo)
	destH := handler.NewDestinationHandler(getDests)
	subH := handler.NewSubscriptionHandler(subscribe, unsubscribe, sendTestPush)
	notifQueueH := handler.NewNotificationQueueHandler(getPendingNotifications)
	inboxH := handler.NewInboxHandler(getInbox, markInboxRead, deleteInboxItem, restoreInboxItem)
	vapidH := handler.NewVapidHandler(vapidKeys.Public)

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
	configH := handler.NewConfigHandler(siteName, returnDelayHours, inboxRetentionDays, digestHour)
	sendMorningDigest := usecase.NewSendMorningDigest(rideRepo, requestRepo, subRepo, notifPrefsRepo, settingsRepo, notifier, serviceTZ, digestHour, siteName)
	notifPrefsH := handler.NewNotificationPreferencesHandler(usecase.NewGetNotificationPreferences(notifPrefsRepo), usecase.NewSetNotificationPreferences(notifPrefsRepo))
	feedbackH := handler.NewFeedbackHandler(recordFeedback)
	statsH := handler.NewStatsHandler(getStats)

	go func() {
		runCronCycle := func() {
			// enqueueFeedback runs FIRST, before expireRides, so an evening ride
			// about to be expired/deleted is still captured into the feedback queue.
			if err := enqueueFeedback.Execute(); err != nil {
				log.Printf("enqueue feedback: %v", err)
			}
			if err := expireRides.Execute(); err != nil {
				log.Printf("expire rides: %v", err)
			}
			if err := expireRequests.Execute(); err != nil {
				log.Printf("expire requests: %v", err)
			}
			if err := expireMatchQueue.Execute(); err != nil {
				log.Printf("expire match queue: %v", err)
			}
			if err := sendFeedbackPrompts.Execute(); err != nil {
				log.Printf("send feedback prompts: %v", err)
			}
			if err := sendAlertExpiryWarnings.Execute(); err != nil {
				log.Printf("alert expiry warnings: %v", err)
			}
			if err := sendMorningDigest.Execute(); err != nil {
				log.Printf("morning digest: %v", err)
			}
			if err := sendEngagementReminder.Execute(); err != nil {
				log.Printf("engagement reminder: %v", err)
			}
			if err := purgeInbox.Execute(); err != nil {
				log.Printf("purge inbox: %v", err)
			}
			// After the purge, so nothing is re-pushed about an expired post.
			if err := sendInboxReminders.Execute(); err != nil {
				log.Printf("inbox reminders: %v", err)
			}
		}
		runCronCycle() // tick at startup — don't wait an hour for the first cycle
		// Then run just after each hour, whenever the app was (re)started, so
		// time-of-day jobs (the 8:00 morning digest, Monday's 8:00 engagement
		// reminder) go out on time instead of up to an hour late.
		time.Sleep(time.Until(time.Now().Truncate(time.Hour).Add(time.Hour + time.Minute)))
		runCronCycle()
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			runCronCycle()
		}
	}()

	r := gin.Default()
	// On Scalingo the real client IP is in X-Real-IP (set by their reverse proxy).
	r.TrustedPlatform = "X-Real-IP"
	// Serve the SvelteKit static build. Any path that is not /api and not an
	// existing file falls back to index.html (client-side routing).
	const buildDir = "./web/build"
	log.Printf("build version: %s", version.Get())
	rideOG := func(id string) (ogRide, bool) {
		ride, err := rideRepo.FindByID(id)
		if err != nil {
			return ogRide{}, false
		}
		return ogRide{Origin: ride.Origin, Destination: ride.Destination, DepartureAt: ride.DepartureAt, Flexibility: int(ride.Flexibility)}, true
	}
	r.NoRoute(spaHandler(buildDir, siteName, rideOG, serviceTZ))

	// SEO endpoints with absolute, host-derived URLs (see og.go). Explicit routes
	// take precedence over the static-file fallback, replacing static robots.txt.
	r.GET("/sitemap.xml", sitemapHandler)
	r.GET("/robots.txt", robotsHandler)

	api := r.Group("/api")
	{
		api.POST("/rides", rideH.Post)
		api.GET("/rides", rideH.List)
		api.GET("/rides/:id", rideH.Get)
		api.PUT("/rides/:id", rideH.Update)
		api.DELETE("/rides/:id", rideH.Delete)
		api.GET("/rides/:id/requests", rideH.ListMatchingRequests)
		api.POST("/rides/:id/feedback", feedbackH.Post)
		api.GET("/rides/:id/interests", rideH.ListInterests)
		api.POST("/rides/:id/interest", interestH.Express)
		api.POST("/interests/:id/accept", interestH.Accept)
		api.DELETE("/interests/:id", interestH.Cancel)
		api.GET("/interests", interestH.ListMyRequests)
		api.GET("/interests/:id/contact", interestH.GetContact)

		api.POST("/requests", reqH.Post)
		api.GET("/requests", reqH.List)
		api.GET("/requests/:id", reqH.Get)
		api.DELETE("/requests/:id", reqH.Delete)
		api.POST("/requests/:id/ping", reqH.Ping)
		api.POST("/requests/:id/offer-contact", reqH.OfferContact)
		api.GET("/requests/:id/offer-contact-status", reqH.GetContactOfferStatus)
		api.GET("/requests/:id/offers", reqH.ListContactOffers)
		api.POST("/requests/:id/renew", reqH.Renew)

		api.GET("/destinations", destH.List)

		api.POST("/subscriptions", subH.Subscribe)
		api.POST("/subscriptions/test", subH.TestPush)
		api.DELETE("/subscriptions/:phone", subH.Unsubscribe)

		api.GET("/notifications", notifQueueH.List)
		api.GET("/inbox", inboxH.List)
		api.GET("/notification-preferences", notifPrefsH.Get)
		api.PUT("/notification-preferences", notifPrefsH.Put)
		api.POST("/inbox/read", inboxH.MarkRead)
		api.DELETE("/inbox/:id", inboxH.Delete)
		api.POST("/inbox/:id/restore", inboxH.Restore)

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

// spaHandler serves the SvelteKit static build from buildDir with an SPA
// fallback: /api/* paths get a JSON 404, existing files are served directly,
// and everything else falls back to index.html for client-side routing.
func spaHandler(buildDir, siteName string, lookup rideLookupFunc, tz *time.Location) gin.HandlerFunc {
	return func(c *gin.Context) {
		p := c.Request.URL.Path
		if strings.HasPrefix(p, "/api/") || p == "/api" {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		clean := filepath.Clean(p)
		file := filepath.Join(buildDir, clean)
		if strings.HasPrefix(file, filepath.Clean(buildDir)+string(os.PathSeparator)) {
			if fi, err := os.Stat(file); err == nil && !fi.IsDir() {
				c.File(file)
				return
			}
			// Prerendered routes (e.g. /about → about.html) are served directly,
			// keeping their own <title>/meta from <svelte:head> (no OG injection).
			if fi, err := os.Stat(file + ".html"); err == nil && !fi.IsDir() {
				c.File(file + ".html")
				return
			}
		}
		// SPA fallback — inject per-page Open Graph tags so crawlers that don't
		// run JS still get a rich link preview.
		indexPath := filepath.Join(buildDir, "index.html")
		shell, err := os.ReadFile(indexPath)
		if err != nil {
			c.File(indexPath) // preserve the prior behaviour (404 if missing)
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(injectOG(string(shell), c, siteName, lookup, tz)))
	}
}
