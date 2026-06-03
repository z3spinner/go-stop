package usecase

import "github.com/z3spinner/go-stop/internal/boundaries/repository"

type CancelInterest struct {
	interests repository.InterestRepository
}

func NewCancelInterest(interests repository.InterestRepository) *CancelInterest {
	return &CancelInterest{interests: interests}
}

// Execute lets a searcher withdraw their own contact request.
//
// Only the searcher who created the interest may cancel it (ErrUnauthorized
// otherwise), and only while it is still pending (ErrNotPending otherwise) —
// once a driver has accepted, contact has been exchanged and there is nothing
// to withdraw. A missing interest surfaces the repository's not-found error.
func (uc *CancelInterest) Execute(interestID, searcherPhone string) error {
	interest, err := uc.interests.FindByID(interestID)
	if err != nil {
		return err
	}
	if interest.SearcherPhone != searcherPhone {
		return ErrUnauthorized
	}
	if interest.Status != "pending" {
		return ErrNotPending
	}
	return uc.interests.Delete(interestID)
}
