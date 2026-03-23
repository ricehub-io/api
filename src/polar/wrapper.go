package polar

import (
	"context"
	"ricehub/src/utils"

	polargo "github.com/polarsource/polar-go"
	"github.com/polarsource/polar-go/models/components"
	"github.com/polarsource/polar-go/models/operations"
	"go.uber.org/zap"
)

var (
	ctx context.Context
	sdk *polargo.Polar
)

func fixedPrice(cents int64) components.ProductPriceFixedCreate {
	return components.ProductPriceFixedCreate{
		PriceCurrency: components.PresentmentCurrencyUsd.ToPointer(),
		PriceAmount:   cents,
	}
}

// Init initializes the global polar variable by creating a new SDK instance
func Init(token string, useSandbox bool) {
	logger := zap.L()

	opts := []polargo.SDKOption{polargo.WithSecurity(token)}

	if useSandbox {
		logger.Warn("Using sandbox server for Polar")
		opts = append(opts, polargo.WithServer(polargo.ServerSandbox))
	}

	ctx = context.Background()
	sdk = polargo.New(opts...)

	logger.Info("Polar has been successfully initialized")
}

// CreateProduct creates new product in Polar with provided name and price.
func CreateProduct(name string, price float64) (res *operations.ProductsCreateResponse, err error) {
	logger := zap.L()

	// convert price to cents
	cents := utils.PriceToCents(price)
	logger.Info(
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
	cents := utils.PriceToCents(newPrice)
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

func CreateCheckoutSession(userID string, riceID string, productID string) (res *operations.CheckoutsCreateResponse, err error) {
	zap.L().Info(
		"Creating a new checkout session",
		zap.String("user_id", userID),
		zap.String("product_id", productID),
	)

	checkout := components.CheckoutCreate{
		ExternalCustomerID: polargo.String(userID),
		Products:           []string{productID},
		EmbedOrigin:        polargo.String(utils.Config.Server.CorsOrigin),
		CustomerBillingAddress: &components.AddressInput{
			Country: components.CountryAlpha2InputUs,
		},
		Metadata: map[string]components.CheckoutCreateMetadata{
			"rice_id": components.CreateCheckoutCreateMetadataStr(riceID),
		},
	}

	res, err = sdk.Checkouts.Create(ctx, checkout)
	return
}
