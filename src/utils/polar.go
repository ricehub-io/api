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

var polar PolarWrapper

// InitPolar initializes the global polar variable by creating a new SDK instance
func InitPolar(token string, useSandbox bool) {
	logger := zap.L()

	opts := []polargo.SDKOption{polargo.WithSecurity(token)}

	if useSandbox {
		logger.Warn("Using sandbox server for Polar")
		opts = append(opts, polargo.WithServer(polargo.ServerSandbox))
	}

	polar.ctx = context.Background()
	polar.sdk = polargo.New(opts...)

	logger.Info("Polar has been successfully initialized")
}

// CreateProduct creates new product in Polar with provided name and price (in cents).
func (w PolarWrapper) CreateProduct(name string, price int64) (res *operations.ProductsCreateResponse, err error) {
	product := components.CreateProductCreateProductCreateOneTime(
		components.ProductCreateOneTime{
			Name: name,
			Prices: []components.ProductCreateOneTimePrices{
				components.CreateProductCreateOneTimePricesFixed(
					components.ProductPriceFixedCreate{
						PriceCurrency: components.PresentmentCurrencyUsd.ToPointer(),
						PriceAmount:   price,
					},
				),
			},
			// Metadata: map[string]components.ProductCreateOneTimeMetadata{
			// 	"dd": components.CreateProductCreateOneTimeMetadataStr("abcd"),
			// },
		},
	)

	res, err = w.sdk.Products.Create(w.ctx, product)
	return
}
