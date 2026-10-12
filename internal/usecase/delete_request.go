// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase

import (
	"log"

	"github.com/z3spinner/go-stop/internal/boundaries/repository"
)

type DeleteRequest struct {
	requests repository.RequestRepository
	inbox    repository.InboxRepository
}

func NewDeleteRequest(requests repository.RequestRepository, inbox repository.InboxRepository) *DeleteRequest {
	return &DeleteRequest{requests: requests, inbox: inbox}
}

func (uc *DeleteRequest) Execute(id, phone string) error {
	req, err := uc.requests.FindByID(id)
	if err != nil {
		return err
	}
	if req.Phone != phone {
		return ErrUnauthorized
	}
	if err := uc.requests.Delete(id); err != nil {
		return err
	}
	// Remove copies of this request (the searcher's name and route) from other
	// people's inboxes right away, as the privacy notice promises. (Expired
	// requests are cleaned by the hourly PurgeInbox.) Best-effort.
	if err := uc.inbox.DeleteForRequest(id); err != nil {
		log.Printf("delete request: inbox cleanup: %v", err)
	}
	return nil
}
