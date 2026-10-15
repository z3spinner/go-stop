// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/z3spinner/go-stop/internal/usecase"
)

type InboxHandler struct {
	getInbox *usecase.GetInbox
	markRead *usecase.MarkInboxRead
	remove   *usecase.DeleteInboxItem
	restore  *usecase.RestoreInboxItem
}

func NewInboxHandler(getInbox *usecase.GetInbox, markRead *usecase.MarkInboxRead, remove *usecase.DeleteInboxItem, restore *usecase.RestoreInboxItem) *InboxHandler {
	return &InboxHandler{getInbox: getInbox, markRead: markRead, remove: remove, restore: restore}
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

// MarkRead marks inbox items as read: the given ids, or every item when all
// is true. Empty ids without all marks nothing.
// @ID       markInboxRead
// @Tags     notifications
// @Accept   json
// @Param    X-Phone  header  string                    true  "Recipient phone"
// @Param    body     body    handler.MarkInboxReadBody  true  "Item ids, or all=true"
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
	if err := h.markRead.Execute(phone, body.IDs, body.All); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// Delete removes one notification from the caller's inbox. Idempotent: an
// unknown, already-deleted or someone else's id also returns 204.
// @ID       deleteInboxItem
// @Tags     notifications
// @Param    id       path    string  true  "Inbox item ID"
// @Param    X-Phone  header  string  true  "Recipient phone"
// @Success  204
// @Failure  401  {object}  handler.ErrorResponse
// @Failure  500  {object}  handler.ErrorResponse
// @Router   /inbox/{id} [delete]
func (h *InboxHandler) Delete(c *gin.Context) {
	phone := normalizePhone(c.GetHeader("X-Phone"))
	if phone == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "X-Phone header required"})
		return
	}
	if err := h.remove.Execute(phone, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// Restore undoes a recent Delete (within 10 minutes).
// @ID       restoreInboxItem
// @Tags     notifications
// @Param    id       path    string  true  "Inbox item ID"
// @Param    X-Phone  header  string  true  "Recipient phone"
// @Success  204
// @Failure  401  {object}  handler.ErrorResponse
// @Failure  404  {object}  handler.ErrorResponse
// @Failure  500  {object}  handler.ErrorResponse
// @Router   /inbox/{id}/restore [post]
func (h *InboxHandler) Restore(c *gin.Context) {
	phone := normalizePhone(c.GetHeader("X-Phone"))
	if phone == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "X-Phone header required"})
		return
	}
	switch err := h.restore.Execute(phone, c.Param("id")); {
	case err == nil:
		c.Status(http.StatusNoContent)
	case errors.Is(err, usecase.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "nothing to restore"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}
