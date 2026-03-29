package config

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/spf13/viper"
)

const (
	appName    = "larm"
	configFile = "config.yml"

	KeyAPIKey       = "api_key"
	KeyAPIURL       = "api_url"
	KeyAccessToken  = "access_token"
	KeyRefreshToken = "refresh_token"
	KeyExpiresAt    = "token_expires_at"
	KeyOrgName      = "organization_name"
)

// ValidKeys is the whitelist of known config keys.
var ValidKeys = []string{
	KeyAPIKey,
	KeyAPIURL,
	KeyAccessToken,
	KeyRefreshToken,
	KeyExpiresAt,
	KeyOrgName,
}

// IsValidKey returns true if the key is in the whitelist.
func IsValidKey(key string) bool {
	return slices.Contains(ValidKeys, key)
}

// Init sets up viper to read from the XDG config file and environment.
func Init() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(configDir())
	viper.SetEnvPrefix("LARM")
	viper.AutomaticEnv()

	// Bind env vars
	_ = viper.BindEnv("api_key")
	_ = viper.BindEnv("api_url")

	// Read config file (ignore if missing)
	_ = viper.ReadInConfig()
}

// Get returns a config value.
func Get(key string) string {
	return viper.GetString(key)
}

// Set writes a config value to the config file.
func Set(key, value string) error {
	dir := configDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	viper.Set(key, value)

	path := filepath.Join(dir, configFile)
	if err := viper.WriteConfigAs(path); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}

// All returns all config keys and values.
func All() map[string]any {
	return viper.AllSettings()
}

// Path returns the config file path.
func Path() string {
	return filepath.Join(configDir(), configFile)
}

func configDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, appName)
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", appName)
}
