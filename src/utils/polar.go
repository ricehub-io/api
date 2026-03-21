package utils

import (
	"context"

	polargo "github.com/polarsource/polar-go"
	"github.com/polarsource/polar-go/models/components"
	"github.com/polarsource/polar-go/models/operations"
	"go.uber.org/zap"
)

type PolarWrapper struct {
	ctx context.Context
	sdk *polargo.Polar
}

var Polar PolarWrapper

// InitPolar initializes the global polar variable by creating a new SDK instance
func InitPolar(token string, useSandbox bool) {
	logger := zap.L()

	opts := []polargo.SDKOption{polargo.WithSecurity(token)}

	if useSandbox {
		logger.Warn("Using sandbox server for Polar")
		opts = append(opts, polargo.WithServer(polargo.ServerSandbox))
	}

	Polar.ctx = context.Background()
	Polar.sdk = polargo.New(opts...)

	logger.Info("Polar has been successfully initialized")
}

func fixedPrice(cents int64) components.ProductPriceFixedCreate {
	return components.ProductPriceFixedCreate{
		PriceCurrency: components.PresentmentCurrencyUsd.ToPointer(),
		PriceAmount:   cents,
	}
}

// CreateProduct creates new product in Polar with provided name and price.
func (w PolarWrapper) CreateProduct(name string, price float32) (res *operations.ProductsCreateResponse, err error) {
	logger := zap.L()

	// convert price to cents
	cents := PriceToCents(price)
	logger.Info(
		"Creating new Polar product for dotfiles",
		zap.String("name", name),
		zap.Float32("price_org", price),
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

	res, err = w.sdk.Products.Create(w.ctx, product)
	return
}

// UpdatePrice updates existing product's price to provided newPrice
func (w PolarWrapper) UpdatePrice(productID string, newPrice float32) (res *operations.ProductsUpdateResponse, err error) {
	cents := PriceToCents(newPrice)
	zap.L().Info(
		"Updating product's price",
		zap.String("product_id", productID),
		zap.Float32("new_price_org", newPrice),
		zap.Int64("new_price_cents", cents),
	)

	update := components.ProductUpdate{
		Prices: []components.ProductUpdatePrices{
			components.CreateProductUpdatePricesTwo(components.CreateTwoFixed(fixedPrice(cents))),
		},
	}

	res, err = w.sdk.Products.Update(w.ctx, productID, update)
	return
}

func (w PolarWrapper) updateVisibility(productID string, newVisibility components.ProductVisibility) (res *operations.ProductsUpdateResponse, err error) {
	zap.L().Info(
		"Updating product's visibility",
		zap.String("product_id", productID),
		zap.String("new_visibility", string(newVisibility)),
	)

	update := components.ProductUpdate{
		Visibility: newVisibility.ToPointer(),
	}

	res, err = w.sdk.Products.Update(w.ctx, productID, update)
	return
}

// ShowProduct changes the visibility of the product to 'public'
func (w PolarWrapper) ShowProduct(productID string) (res *operations.ProductsUpdateResponse, err error) {
	return w.updateVisibility(productID, components.ProductVisibilityPublic)
}

// HideProduct changes the visibility of the product to 'private'
func (w PolarWrapper) HideProduct(productID string) (res *operations.ProductsUpdateResponse, err error) {
	return w.updateVisibility(productID, components.ProductVisibilityPrivate)
}

func (w PolarWrapper) ArchiveProduct(productID string) (res *operations.ProductsUpdateResponse, err error) {
	zap.L().Info(
		"Archiving product",
		zap.String("product_id", productID),
	)

	update := components.ProductUpdate{
		IsArchived: polargo.Bool(true),
	}

	res, err = w.sdk.Products.Update(w.ctx, productID, update)
	return
}
