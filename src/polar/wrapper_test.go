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
