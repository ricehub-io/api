package utils

import "math"

// Try to construct URL from avatar path
// or use the default one if user didn't set any
func GetUserAvatar(avatarPath *string) string {
	cdnURL := Config.App.CDNUrl

	avatar := cdnURL + Config.App.DefaultAvatar
	if avatarPath != nil {
		avatar = cdnURL + *avatarPath
	}
	return avatar
}

// PriceToCents converts price in normal format (e.g. 15.89) to cents (in this case 1589) and returns them.
func PriceToCents(price float64) int64 {
	return int64(math.RoundToEven(price * 100.0))
}
