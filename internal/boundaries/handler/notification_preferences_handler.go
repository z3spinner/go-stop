// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/usecase"
)

type NotificationPreferencesHandler struct {
	get *usecase.GetNotificationPreferences
	set *usecase.SetNotificationPreferences
}

func NewNotificationPreferencesHandler(get *usecase.GetNotificationPreferences, set *usecase.SetNotificationPreferences) *NotificationPreferencesHandler {
	return &NotificationPreferencesHandler{get: get, set: set}
}

func toPreferencesDTO(p domain.NotificationPreferences) NotificationPreferences {
	return NotificationPreferences{MorningDigest: p.MorningDigest}
}

// Get returns the caller's notification preferences (defaults: everything on).
// @ID       getNotificationPreferences
// @Tags     notifications
// @Produce  json
// @Param    X-Phone  header  string  true  "Phone"
// @Success  200  {object}  handler.NotificationPreferences
// @Failure  401  {object}  handler.ErrorResponse
// @Failure  500  {object}  handler.ErrorResponse
// @Router   /notification-preferences [get]
func (h *NotificationPreferencesHandler) Get(c *gin.Context) {
	phone := normalizePhone(c.GetHeader("X-Phone"))
	if phone == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "X-Phone header required"})
		return
	}
	p, err := h.get.Execute(phone)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toPreferencesDTO(p))
}

// Put saves the caller's notification preferences.
// @ID       setNotificationPreferences
// @Tags     notifications
// @Accept   json
// @Produce  json
// @Param    X-Phone  header  string                           true  "Phone"
// @Param    body     body    handler.NotificationPreferences  true  "Preferences"
// @Success  200  {object}  handler.NotificationPreferences
// @Failure  400  {object}  handler.ErrorResponse
// @Failure  401  {object}  handler.ErrorResponse
// @Failure  500  {object}  handler.ErrorResponse
// @Router   /notification-preferences [put]
func (h *NotificationPreferencesHandler) Put(c *gin.Context) {
	phone := normalizePhone(c.GetHeader("X-Phone"))
	if phone == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "X-Phone header required"})
		return
	}
	var body NotificationPreferences
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	p, err := h.set.Execute(phone, domain.NotificationPreferences{MorningDigest: body.MorningDigest})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toPreferencesDTO(p))
}
