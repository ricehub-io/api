package polar

import (
	"testing"

	"github.com/polarsource/polar-go/models/components"
)

// #################################################
// ################# fixedPrice ####################
// #################################################
func TestFixedPrice_SetsAmountCorrectly(t *testing.T) {
	p := fixedPrice(1589)
	if p.PriceAmount != 1589 {
		t.Errorf("want PriceAmount=1589, got %d", p.PriceAmount)
	}
}

func TestFixedPrice_CurrencyIsUSD(t *testing.T) {
	p := fixedPrice(1000)
	if p.PriceCurrency == nil {
		t.Fatal("expected PriceCurrency to be set, got nil")
	}
	if *p.PriceCurrency != components.PresentmentCurrencyUsd {
		t.Errorf("want USD, got %v", *p.PriceCurrency)
	}
}

func TestFixedPrice_ZeroCents(t *testing.T) {
	p := fixedPrice(0)
	if p.PriceAmount != 0 {
		t.Errorf("want 0, got %d", p.PriceAmount)
	}
}

func TestFixedPrice_LargeAmount(t *testing.T) {
	p := fixedPrice(999999)
	if p.PriceAmount != 999999 {
		t.Errorf("want 999999, got %d", p.PriceAmount)
	}
}

// #################################################
// ################# priceToCents ##################
// #################################################
func TestPriceToCents_WholeNumber(t *testing.T) {
	if got := priceToCents(15); got != 1500 {
		t.Errorf("want 1500, got %d", got)
	}
}

func TestPriceToCents_TwoDecimalPlaces(t *testing.T) {
	if got := priceToCents(15.89); got != 1589 {
		t.Errorf("want 1589, got %d", got)
	}
}

func TestPriceToCents_OneDecimalPlace(t *testing.T) {
	if got := priceToCents(9.9); got != 990 {
		t.Errorf("want 990, got %d", got)
	}
}

func TestPriceToCents_Zero(t *testing.T) {
	if got := priceToCents(0); got != 0 {
		t.Errorf("want 0, got %d", got)
	}
}

func TestPriceToCents_RoundsHalfUp(t *testing.T) {
	// 10.005 * 100 = 1000.5 therefore it should
	// round to 1001, not truncate to 1000
	if got := priceToCents(10.005); got != 1001 {
		t.Errorf("want 1001, got %d", got)
	}
}

func TestPriceToCents_FloatingPointImprecision(t *testing.T) {
	// 2.675 float number has some issues due to binary representation
	// this makes sure that the rounding is correct
	// TIL https://github.com/dotnet/runtime/issues/121112#issuecomment-3451957155
	if got := priceToCents(2.675); got != 268 {
		t.Errorf("want 268, got %d", got)
	}
}

func TestPriceToCents_LargePrice(t *testing.T) {
	if got := priceToCents(9999.99); got != 999999 {
		t.Errorf("want 999999, got %d", got)
	}
}
