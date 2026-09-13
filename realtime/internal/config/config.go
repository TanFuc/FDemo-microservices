package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port      int
	NatsURL   string
	JWTSecret string
}

func Load() *Config {
	port := 3015
	if p := os.Getenv("SERVER_PORT"); p != "" {
		if val, err := strconv.Atoi(p); err == nil {
			port = val
		}
	} else if p := os.Getenv("PORT"); p != "" {
		if val, err := strconv.Atoi(p); err == nil {
			port = val
		}
	}

	natsURL := "nats://localhost:4222"
	if u := os.Getenv("NATS_URL"); u != "" {
		natsURL = u
	}

	jwtSecret := "nexus_enterprise_super_secret_jwt_access_token_32chars_min"
	if s := os.Getenv("JWT_SECRET"); s != "" {
		jwtSecret = s
	} else if s := os.Getenv("JWT_ACCESS_SECRET"); s != "" {
		jwtSecret = s
	}

	return &Config{
		Port:      port,
		NatsURL:   natsURL,
		JWTSecret: jwtSecret,
	}
}
