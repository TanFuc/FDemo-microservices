package config

import (
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Services  ServicesConfig  `yaml:"services"`
	Redis     RedisConfig     `yaml:"redis"`
	JWT       JWTConfig       `yaml:"jwt"`
	RateLimit RateLimitConfig `yaml:"rate_limit"`
	Internal  InternalConfig  `yaml:"internal"`
}

type ServerConfig struct {
	Port    int  `yaml:"port"`
	Prefork bool `yaml:"prefork"`
}

type ServicesConfig struct {
	CatalogURL      string `yaml:"catalog_url"`
	CartURL         string `yaml:"cart_url"`
	OrderURL        string `yaml:"order_url"`
	IdentityURL     string `yaml:"identity_url"`
	ProfileURL      string `yaml:"profile_url"`
	PaymentURL      string `yaml:"payment_url"`
	LogisticURL     string `yaml:"logistic_url"`
	MediaURL        string `yaml:"media_url"`
	ReviewURL       string `yaml:"review_url"`
	SearchURL       string `yaml:"search_url"`
	CampaignURL     string `yaml:"campaign_url"`
	AnalyticURL     string `yaml:"analytic_url"`
	InventoryURL    string `yaml:"inventory_url"`
	NotificationURL string `yaml:"notification_url"`
	RealtimeURL     string `yaml:"realtime_url"`
}

type InternalConfig struct {
	ServiceKey string `yaml:"service_key"`
}

type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type JWTConfig struct {
	Secret string `yaml:"secret"`
	Issuer string `yaml:"issuer"`
}

type RateLimitConfig struct {
	MaxRequests       int `yaml:"max_requests"`
	ExpirationSeconds int `yaml:"expiration_seconds"`
}

func Load(path string) (*Config, error) {
	loadDotEnv(".env")
	loadDotEnv("../.env")

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	cfg.applyEnvOverrides()

	return &cfg, nil
}

func loadDotEnv(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			k := strings.TrimSpace(parts[0])
			v := strings.TrimSpace(parts[1])
			v = strings.Trim(v, "\"'")
			if os.Getenv(k) == "" {
				os.Setenv(k, v)
			}
		}
	}
}

func (c *Config) applyEnvOverrides() {
	if p := os.Getenv("PORT"); p != "" {
		if val, err := strconv.Atoi(p); err == nil {
			c.Server.Port = val
		}
	} else if p := os.Getenv("SERVER_PORT"); p != "" {
		if val, err := strconv.Atoi(p); err == nil {
			c.Server.Port = val
		}
	}
	if v := os.Getenv("CATALOG_URL"); v != "" {
		c.Services.CatalogURL = v
	}
	if v := os.Getenv("CART_URL"); v != "" {
		c.Services.CartURL = v
	}
	if v := os.Getenv("ORDER_URL"); v != "" {
		c.Services.OrderURL = v
	}
	if v := os.Getenv("IDENTITY_URL"); v != "" {
		c.Services.IdentityURL = v
	}
	if v := os.Getenv("PROFILE_URL"); v != "" {
		c.Services.ProfileURL = v
	}
	if v := os.Getenv("PAYMENT_URL"); v != "" {
		c.Services.PaymentURL = v
	}
	if v := os.Getenv("LOGISTIC_URL"); v != "" {
		c.Services.LogisticURL = v
	}
	if v := os.Getenv("MEDIA_URL"); v != "" {
		c.Services.MediaURL = v
	}
	if v := os.Getenv("REVIEW_URL"); v != "" {
		c.Services.ReviewURL = v
	}
	if v := os.Getenv("SEARCH_URL"); v != "" {
		c.Services.SearchURL = v
	}
	if v := os.Getenv("CAMPAIGN_URL"); v != "" {
		c.Services.CampaignURL = v
	}
	if v := os.Getenv("ANALYTIC_URL"); v != "" {
		c.Services.AnalyticURL = v
	}
	if v := os.Getenv("INVENTORY_URL"); v != "" {
		c.Services.InventoryURL = v
	}
	if v := os.Getenv("NOTIFICATION_URL"); v != "" {
		c.Services.NotificationURL = v
	}
	if v := os.Getenv("REALTIME_URL"); v != "" {
		c.Services.RealtimeURL = v
	}
	if v := os.Getenv("REDIS_HOST"); v != "" {
		c.Redis.Host = v
	}
	if v := os.Getenv("REDIS_PORT"); v != "" {
		if val, err := strconv.Atoi(v); err == nil {
			c.Redis.Port = val
		}
	}
	if v := os.Getenv("REDIS_PASSWORD"); v != "" {
		c.Redis.Password = v
	}
	if v := os.Getenv("JWT_SECRET"); v != "" {
		c.JWT.Secret = v
	}
	if v := os.Getenv("INTERNAL_SERVICE_KEY"); v != "" {
		c.Internal.ServiceKey = v
	}
}
