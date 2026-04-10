package config

import (
	"time"

	"github.com/BurntSushi/toml"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type (
	rootConfig struct {
		Server    serverConfig
		Database  databaseConfig
		App       appConfig
		JWT       jwtConfig
		Polar     polarConfig
		Limits    limitsConfig
		Blacklist blacklistConfig
	}

	serverConfig struct {
		Port          uint16 `toml:"port"`
		CorsOrigin    string `toml:"cors_origin"`
		CookiesDomain string `toml:"cookies_domain"`
		KeysDir       string `toml:"keys_dir"`
	}

	databaseConfig struct {
		DatabaseUrl string `toml:"database_url"`
		RedisUrl    string `toml:"redis_url"`
	}

	appConfig struct {
		CDNUrl            string `toml:"cdn_url"`
		DefaultAvatar     string `toml:"default_avatar"`
		PaginationLimit   uint   `toml:"pagination_limit"`
		DisableRateLimits bool   `toml:"disable_rate_limits"`
		Maintenance       bool   `toml:"maintenance"`
	}

	jwtConfig struct {
		AccessExpiration  time.Duration `toml:"access_exp"`
		RefreshExpiration time.Duration `toml:"refresh_exp"`
	}

	polarConfig struct {
		Sandbox               bool      `toml:"sandbox"`
		Token                 string    `toml:"token"`
		WebhookSecret         string    `toml:"webhook_secret"`
		SubscriptionProductID uuid.UUID `toml:"subscription_product_id"`
	}

	limitsConfig struct {
		MaxScreenshotsPerRice int64 `toml:"max_screenshots_per_rice"`
		UserAvatarSizeLimit   int64 `toml:"user_avatar_size_limit"`
		DotfilesSizeLimit     int64 `toml:"dotfiles_size_limit"`
		ScreenshotSizeLimit   int64 `toml:"screenshot_size_limit"`
	}

	blacklistConfig struct {
		Words        []string `toml:"words"`
		DisplayNames []string `toml:"display_names"`
		Usernames    []string `toml:"usernames"`
	}
)

var Config rootConfig

func InitConfig(configPath string) {
	log := zap.L()
	log.Info(
		"Reading config file...",
		zap.String("path", configPath),
	)

	_, err := toml.DecodeFile(configPath, &Config)
	if err != nil {
		log.Fatal("Failed to decode config file", zap.Error(err))
	}

	if Config.Database.DatabaseUrl == "" || Config.Database.RedisUrl == "" || Config.Server.Port == 0 {
		log.Fatal("Missing required config fields (database.database_url, database.redis_url, server.port)")
	}

	if Config.Limits.MaxScreenshotsPerRice <= 0 {
		log.Fatal("limits.max_screenshots_per_rice must be greater than zero")
	}

	log.Info("Config variables successfully loaded")
}
