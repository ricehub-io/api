package services

import (
	"ricehub/internal/errs"
	"ricehub/internal/models"
	"ricehub/internal/polar"
	"ricehub/internal/repository"

	"github.com/google/uuid"
)

type LinkService struct{}

func NewLinkService() *LinkService {
	return &LinkService{}
}

// GetLinkByName fetches a link by its name.
// Returns an error if no link with the given name exists.
func (s *LinkService) GetLinkByName(name string) (models.Link, errs.AppError) {
	link, err := repository.FindLink(name)
	if err != nil {
		return link, errs.FromDBError(err, errs.LinkNotFound)
	}
	return link, nil
}

// GetSubscriptionLink checks if user has an active subscription and returns a Polar checkout URL.
// Returns an error if the user already has an active subscription.
func (s *LinkService) GetSubscriptionLink(userID, productID uuid.UUID) (string, errs.AppError) {
	subActive, err := repository.SubscriptionActive(userID)
	if err != nil {
		return "", errs.InternalError(err)
	}
	if subActive {
		return "", errs.ActiveSubscription
	}

	res, err := polar.CreateCheckoutSession(userID, productID)
	if err != nil {
		return "", errs.InternalError(err)
	}

	return res.Checkout.URL, nil
}
