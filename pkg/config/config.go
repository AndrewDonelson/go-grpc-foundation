package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Load loads configuration from file and environment
func Load(configPath string, configName string) error {
	viper.SetConfigName(configName)
	viper.SetConfigType("yaml")
	viper.AddConfigPath(configPath)
	viper.AddConfigPath(".")

	// Enable environment variable override
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found, use environment variables only
			return nil
		}
		return fmt.Errorf("failed to read config: %w", err)
	}

	return nil
}

// GetString gets a string config value
func GetString(key string) string {
	return viper.GetString(key)
}

// GetInt gets an int config value
func GetInt(key string) int {
	return viper.GetInt(key)
}

// GetBool gets a bool config value
func GetBool(key string) bool {
	return viper.GetBool(key)
}

// GetDuration gets a duration config value
func GetDuration(key string) interface{} {
	return viper.Get(key)
}
