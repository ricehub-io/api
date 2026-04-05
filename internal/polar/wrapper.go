package polar

import (
	"context"
	"fmt"
	"math"
	"ricehub/internal/config"
	"time"

	"github.com/google/uuid"
	polargo "github.com/polarsource/polar-go"
	"github.com/polarsource/polar-go/models/components"
	"github.com/polarsource/polar-go/models/operations"
	"go.uber.org/zap"
)

var (
	ctx context.Context
	sdk *polargo.Polar
)

// Init initializes the global polar variable by creating a new SDK instance
func Init(token string, useSandbox bool) {
	l := zap.L()

	opts := []polargo.SDKOption{polargo.WithSecurity(token)}

	if useSandbox {
		l.Warn("Using sandbox server for Polar")
		opts = append(opts, polargo.WithServer(polargo.ServerSandbox))
	}

	ctx = context.Background()
	sdk = polargo.New(opts...)

	l.Info("Polar has been successfully initialized")
}

// CreateProduct creates new product in Polar with provided name and price.
func CreateProduct(name string, price float64) (res *operations.ProductsCreateResponse, err error) {
	polarCheck()
	l := zap.L()

	// convert price to cents
	cents := priceToCents(price)
	l.Info(
		"Creating new Polar product for dotfiles",
		zap.String("name", name),
		zap.Float64("price_org", price),
		zap.Int64("price_cents", cents),
	)

	// create new polar product using SDK
	product := components.CreateProductCreateProductCreateOneTime(
		components.ProductCreateOneTime{
			Name: name,
			Prices: []components.ProductCreateOneTimePrices{
				components.CreateProductCreateOneTimePricesFixed(fixedPrice(cents)),
			},
		},
	)

	res, err = sdk.Products.Create(ctx, product)
	return
}

// UpdatePrice updates existing product's price to provided newPrice
func UpdatePrice(productID string, newPrice float64) (res *operations.ProductsUpdateResponse, err error) {
	polarCheck()

	cents := priceToCents(newPrice)
	zap.L().Info(
		"Updating product's price",
		zap.String("product_id", productID),
		zap.Float64("new_price_org", newPrice),
		zap.Int64("new_price_cents", cents),
	)

	update := components.ProductUpdate{
		Prices: []components.ProductUpdatePrices{
			components.CreateProductUpdatePricesTwo(components.CreateTwoFixed(fixedPrice(cents))),
		},
	}

	res, err = sdk.Products.Update(ctx, productID, update)
	return
}

func updateVisibility(productID string, newVisibility components.ProductVisibility) (res *operations.ProductsUpdateResponse, err error) {
	polarCheck()
	zap.L().Info(
		"Updating product's visibility",
		zap.String("product_id", productID),
		zap.String("new_visibility", string(newVisibility)),
	)

	update := components.ProductUpdate{
		Visibility: newVisibility.ToPointer(),
	}

	res, err = sdk.Products.Update(ctx, productID, update)
	return
}

// ShowProduct changes the visibility of the product to 'public'
func ShowProduct(productID string) (res *operations.ProductsUpdateResponse, err error) {
	return updateVisibility(productID, components.ProductVisibilityPublic)
}

// HideProduct changes the visibility of the product to 'private'
func HideProduct(productID string) (res *operations.ProductsUpdateResponse, err error) {
	return updateVisibility(productID, components.ProductVisibilityPrivate)
}

func ArchiveProduct(productID string) (res *operations.ProductsUpdateResponse, err error) {
	polarCheck()

	zap.L().Info(
		"Archiving product",
		zap.String("product_id", productID),
	)

	update := components.ProductUpdate{
		IsArchived: polargo.Bool(true),
	}

	res, err = sdk.Products.Update(ctx, productID, update)
	return
}

func CreateCheckoutSession(userID string, productID uuid.UUID) (res *operations.CheckoutsCreateResponse, err error) {
	zap.L().Info(
		"Creating a new checkout session",
		zap.String("user_id", userID),
		zap.String("product_id", productID.String()),
	)

	checkout := components.CheckoutCreate{
		ExternalCustomerID: polargo.String(userID),
		Products:           []string{productID.String()},
		EmbedOrigin:        polargo.String(config.Config.Server.CorsOrigin),
		CustomerBillingAddress: &components.AddressInput{
			Country: components.CountryAlpha2InputUs,
		},
	}

	res, err = sdk.Checkouts.Create(ctx, checkout)
	return
}

func EventList(eventType components.SystemEventType, eventsAfter *time.Time) (events []components.SystemEvent, err error) {
	polarCheck()

	eventName := string(eventType)
	zap.L().Info("Fetching Polar's event list",
		zap.String("event_name", eventName),
		zap.Timep("events_after", eventsAfter),
	)

	request := operations.EventsListRequest{
		Name: &operations.NameFilter{
			Str: polargo.String(eventName),
		},
		StartTimestamp: eventsAfter,
	}

	res, err := sdk.Events.List(ctx, request)
	if err != nil {
		return
	}

	respList := res.GetResponseEventsList()
	if respList == nil {
		return events, fmt.Errorf("response events list is nil")
	}

	resList := respList.ListResourceEvent
	if resList == nil {
		return events, fmt.Errorf("resource event list is nil")
	}
	allEvents := resList.Items

	// Make sure that only given event type is in the list
	// because later on the function user needs to do some
	// shenanigans with nullable fields to get the actual event data
	// unaimeds: ik it might not be the best performance-wise but its called twice a day.
	for _, ev := range allEvents {
		if ev.Type != components.EventUnionTypeSystemEvent || ev.SystemEvent.Type != eventType {
			continue
		}
		events = append(events, *ev.SystemEvent)
	}

	return
}

func SubscriptionList() (subs []components.Subscription, err error) {
	polarCheck()

	res, err := sdk.Subscriptions.List(ctx, operations.SubscriptionsListRequest{
		Active: polargo.Bool(true),
		Limit:  polargo.Int64(100),
	})
	if err != nil || res.ListResourceSubscription == nil {
		return
	}

	for {
		subs = append(subs, res.ListResourceSubscription.Items...)

		// go to next page
		res, err = res.Next()
		if err != nil {
			return
		}

		// end of list
		if res == nil {
			break
		}
	}

	return
}

// polarCheck checks if Polar SDK has been initialized.
func polarCheck() {
	if sdk == nil {
		zap.L().Fatal("Polar not initialized!")
	}
}

// priceToCents converts price in normal format (e.g. 15.89) to cents (in this case 1589) and returns them.
func priceToCents(price float64) int64 {
	return int64(math.RoundToEven(price * 100.0))
}

func fixedPrice(cents int64) components.ProductPriceFixedCreate {
	return components.ProductPriceFixedCreate{
		PriceCurrency: components.PresentmentCurrencyUsd.ToPointer(),
		PriceAmount:   cents,
	}
}
