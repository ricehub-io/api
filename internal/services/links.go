package services

import (
	"context"

	"github.com/ricehub-io/api/internal/errs"
	"github.com/ricehub-io/api/internal/models"
	"github.com/ricehub-io/api/internal/repository"

	"github.com/google/uuid"
)

type LinkService struct {
	links *repository.LinkRepository
	users *repository.UserRepository
	bans  *repository.UserBanRepository
}

func NewLinkService(
	links *repository.LinkRepository,
	users *repository.UserRepository,
	bans *repository.UserBanRepository,
) *LinkService {
	return &LinkService{links, users, bans}
}

// GetLinkByName fetches a link by its name.
// Returns an error if no link with the given name exists.
func (s *LinkService) GetLinkByName(ctx context.Context, name string) (models.Link, errs.AppError) {
	link, err := s.links.FindLink(ctx, name)
	if err != nil {
		return link, errs.FromDBError(err, errs.LinkNotFound)
	}
	return link, nil
}

// GetSubscriptionLink checks if user has an active subscription and returns a Polar checkout URL.
// Returns an error if the user already has an active subscription.
func (s *LinkService) GetSubscriptionLink(
	ctx context.Context,
	userID, productID uuid.UUID,
) (string, errs.AppError) {
	// TODO: call payments service via gRPC client
	return "", nil
	// if _, err := security.VerifyUserID(ctx, s.users, s.bans, userID.String()); err != nil {
	// 	return "", err
	// }

	// subActive, err := s.userSubs.SubscriptionActive(ctx, userID)
	// if err != nil {
	// 	return "", errs.InternalError(err)
	// }
	// if subActive {
	// 	return "", errs.ActiveSubscription
	// }

	// res, err := polar.CreateCheckoutSession(userID, productID)
	// if err != nil {
	// 	return "", errs.InternalError(err)
	// }

	// return res.Checkout.URL, nil
}
