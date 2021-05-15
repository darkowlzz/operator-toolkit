package export

import (
	"os"
	"strconv"
)

const (
	// Whether the exporter is disabled or not.
	envDisableTracing = "DISABLE_TRACING"
)

// getEnv returns environment variable value for a given key. If the variable
// isn't set, it returns the default value.
func getEnv(key string, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}

	return defaultVal
}

// getEnvAsBool returns boolean value of an environment variable for a given
// key, with a default value if not set.
func getEnvAsBool(name string, defaultVal bool) bool {
	valStr := getEnv(name, "")
	if val, err := strconv.ParseBool(valStr); err == nil {
		return val
	}

	return defaultVal
}
