// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/z3spinner/go-stop/internal/usecase"
)

type InboxHandler struct {
	getInbox *usecase.GetInbox
	markRead *usecase.MarkInboxRead
}

func NewInboxHandler(getInbox *usecase.GetInbox, markRead *usecase.MarkInboxRead) *InboxHandler {
	return &InboxHandler{getInbox: getInbox, markRead: markRead}
}

// List returns the caller's in-app inbox (newest first) and unread count.
// Items are kept NOTIFICATION_RETENTION_DAYS (default 30) days.
// @ID       listInbox
// @Tags     notifications
// @Produce  json
// @Param    X-Phone  header  string  true  "Recipient phone"
// @Success  200  {object}  handler.InboxResponse
// @Failure  401  {object}  handler.ErrorResponse
// @Failure  500  {object}  handler.ErrorResponse
// @Router   /inbox [get]
func (h *InboxHandler) List(c *gin.Context) {
	phone := normalizePhone(c.GetHeader("X-Phone"))
	if phone == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "X-Phone header required"})
		return
	}
	inbox, err := h.getInbox.Execute(phone)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := InboxResponse{Items: make([]InboxItem, len(inbox.Items)), Unread: inbox.Unread}
	for i, it := range inbox.Items {
		item := InboxItem{
			ID:          it.ID,
			Kind:        it.Kind,
			ContactName: it.ContactName,
			Origin:      it.Origin,
			Destination: it.Destination,
			URL:         it.URL,
			Title:       it.Title,
			Body:        it.Body,
			CreatedAt:   it.CreatedAt.UTC().Format(time.RFC3339),
			Read:        !it.ReadAt.IsZero(),
		}
		if !it.DepartureAt.IsZero() {
			item.DepartureAt = it.DepartureAt.UTC().Format(time.RFC3339)
		}
		out.Items[i] = item
	}
	c.JSON(http.StatusOK, out)
}

// MarkRead marks inbox items as read: the given ids, or all when ids is empty.
// @ID       markInboxRead
// @Tags     notifications
// @Accept   json
// @Param    X-Phone  header  string                    true  "Recipient phone"
// @Param    body     body    handler.MarkInboxReadBody  true  "Item ids (empty = all)"
// @Success  204
// @Failure  400  {object}  handler.ErrorResponse
// @Failure  401  {object}  handler.ErrorResponse
// @Failure  500  {object}  handler.ErrorResponse
// @Router   /inbox/read [post]
func (h *InboxHandler) MarkRead(c *gin.Context) {
	phone := normalizePhone(c.GetHeader("X-Phone"))
	if phone == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "X-Phone header required"})
		return
	}
	var body MarkInboxReadBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.markRead.Execute(phone, body.IDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
