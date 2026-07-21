package config

import (
	"crypto/rand"
	"flag"
	"os"
	"strings"
)

// Config holds all configuration for the service.
type Config struct {
	Addr   string
	Data   string
	Secret string
	SMTP   string
}

// Load reads configuration from defaults, env vars, and flags.
// Priority: defaults < env vars < flags.
func Load() *Config {
	c := &Config{
		Addr:   ":7700",
		Data:   "./contactkit-data.json",
		Secret: "",
		SMTP:   "",
	}

	// Env vars
	if v := os.Getenv("CONTACTKIT_ADDR"); v != "" {
		c.Addr = v
	}
	if v := os.Getenv("CONTACTKIT_DATA"); v != "" {
		c.Data = v
	}
	if v := os.Getenv("CONTACTKIT_SECRET"); v != "" {
		c.Secret = v
	}
	if v := os.Getenv("CONTACTKIT_SMTP"); v != "" {
		c.SMTP = v
	}

	// Flags
	flag.StringVar(&c.Addr, "addr", c.Addr, "listen address")
	flag.StringVar(&c.Data, "data", c.Data, "data file path")
	flag.StringVar(&c.Secret, "secret", c.Secret, "token signing secret (auto-generated if empty)")
	flag.StringVar(&c.SMTP, "smtp", c.SMTP, "SMTP server for OTP emails (host:port, empty = log to stderr)")
	flag.Parse()

	// Auto-generate secret if not provided
	if c.Secret == "" {
		c.Secret = randomSecret()
	}

	return c
}

// Sanitize ensures the data path is clean.
func (c *Config) Sanitize() {
	c.Data = strings.TrimSpace(c.Data)
	c.Addr = strings.TrimSpace(c.Addr)
}

func randomSecret() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// Fallback: this should never happen, but if it does we still need a secret
		return "fallback-secret-change-me-in-production-please"
	}
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	for i := range b {
		b[i] = chars[int(b[i])%len(chars)]
	}
	return string(b)
}
