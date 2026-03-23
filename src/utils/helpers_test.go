package utils

import "testing"

func init() {
	// init config values for our tests
	Config.App.CDNUrl = "https://cdn.example.com"
	Config.App.DefaultAvatar = "/avatars/default.png"
}

// #################################################
// ################# GetUserAvatar #################
// #################################################
func TestGetUserAvatar_NilPath_UsesDefault(t *testing.T) {
	got := GetUserAvatar(nil)
	want := "https://cdn.example.com/avatars/default.png"
	if got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}

func TestGetUserAvatar_WithPath_UsesProvidedPath(t *testing.T) {
	path := "/avatars/user-123.png"
	got := GetUserAvatar(&path)
	want := "https://cdn.example.com/avatars/user-123.png"
	if got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}

func TestGetUserAvatar_EmptyCDNUrl(t *testing.T) {
	original := Config
	defer func() { Config = original }()
	Config.App.CDNUrl = ""

	got := GetUserAvatar(nil)
	if got != "/avatars/default.png" {
		t.Errorf("want %q, got %q", "/avatars/default.png", got)
	}
}

func TestGetUserAvatar_ProvidedPathIgnoresDefault(t *testing.T) {
	path := "/avatars/custom.png"
	got := GetUserAvatar(&path)

	// result must not contain the default avatar path
	if got == "https://cdn.example.com/avatars/default.png" {
		t.Error("expected custom path, got default avatar instead")
	}
}

// #################################################
// ################# PriceToCents ##################
// #################################################
func TestPriceToCents_WholeNumber(t *testing.T) {
	if got := PriceToCents(15); got != 1500 {
		t.Errorf("want 1500, got %d", got)
	}
}

func TestPriceToCents_TwoDecimalPlaces(t *testing.T) {
	if got := PriceToCents(15.89); got != 1589 {
		t.Errorf("want 1589, got %d", got)
	}
}

func TestPriceToCents_OneDecimalPlace(t *testing.T) {
	if got := PriceToCents(9.9); got != 990 {
		t.Errorf("want 990, got %d", got)
	}
}

func TestPriceToCents_Zero(t *testing.T) {
	if got := PriceToCents(0); got != 0 {
		t.Errorf("want 0, got %d", got)
	}
}

func TestPriceToCents_RoundsHalfUp(t *testing.T) {
	// 10.005 * 100 = 1000.5 therefore it should
	// round to 1001, not truncate to 1000
	if got := PriceToCents(10.005); got != 1001 {
		t.Errorf("want 1001, got %d", got)
	}
}

func TestPriceToCents_FloatingPointImprecision(t *testing.T) {
	// 2.675 float number has some issues due to binary representation
	// this makes sure that the rounding is correct
	// TIL https://github.com/dotnet/runtime/issues/121112#issuecomment-3451957155
	if got := PriceToCents(2.675); got != 268 {
		t.Errorf("want 268, got %d", got)
	}
}

func TestPriceToCents_LargePrice(t *testing.T) {
	if got := PriceToCents(9999.99); got != 999999 {
		t.Errorf("want 999999, got %d", got)
	}
}
