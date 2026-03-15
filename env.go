package ok

import (
	"os"
	"strconv"
	"time"
)

// Environment variable names for ok configuration.
const (
	EnvBaseURL            = "OK_BASE_URL"
	EnvInsecureSkipVerify = "OK_INSECURE_SKIP_VERIFY"
	EnvTimeout            = "OK_TIMEOUT"
	EnvRetryDisable       = "OK_RETRY_DISABLE"
)

// EnableEnvValues is a package-level flag to enable environment variable
// reading. When set to true, environment variables will be consulted.
// By default, environment variable reading is disabled.
var EnableEnvValues bool

// applyEnvValues reads environment variables and applies them to the option
// value. Environment variables take lower precedence than explicitly set options.
// Only applied when env values are enabled (via EnableEnvValues or WithEnableEnvValues).
func applyEnvValues(o *optionClientValue) {
	if !EnableEnvValues && !o.EnableEnvValues {
		return
	}

	if v := os.Getenv(EnvBaseURL); v != "" && o.BaseURL == "" {
		o.BaseURL = v
	}

	if v := os.Getenv(EnvInsecureSkipVerify); v != "" {
		if b, err := strconv.ParseBool(v); err == nil && b {
			o.InsecureSkipVerify = true
		}
	}

	if v := os.Getenv(EnvTimeout); v != "" && o.Timeout == 0 {
		if d, err := time.ParseDuration(v); err == nil {
			o.Timeout = d
		}
	}

	if v := os.Getenv(EnvRetryDisable); v != "" {
		if b, err := strconv.ParseBool(v); err == nil && b {
			o.DisableRetry = true
		}
	}
}
