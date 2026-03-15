package ok

import "time"

// Config is a declarative configuration struct for building a Client.
// It can be populated from configuration files, environment, or manually.
// Use ToOption() to convert it to a functional option, or New() to directly
// create a Client.
type Config struct {
	// BaseURL is the base URL for all requests.
	BaseURL string `cfg:"base_url" json:"base_url,omitempty"`

	// Header contains default headers applied to all requests.
	Header map[string][]string `cfg:"header" json:"header,omitempty"`

	// Timeout is the overall HTTP client timeout.
	Timeout time.Duration `cfg:"timeout" json:"timeout,omitempty"`

	// EnableBaseURLCheck enables validation of the base URL.
	// By default, base URL validation is disabled.
	EnableBaseURLCheck *bool `cfg:"enable_base_url_check" json:"enable_base_url_check,omitempty"`

	// EnableEnvValues enables reading configuration from environment variables.
	// By default, environment variable reading is disabled.
	EnableEnvValues *bool `cfg:"enable_env_values" json:"enable_env_values,omitempty"`

	// InsecureSkipVerify disables TLS certificate verification.
	InsecureSkipVerify *bool `cfg:"insecure_skip_verify" json:"insecure_skip_verify,omitempty"`

	// DisableRetry disables automatic retry behavior.
	DisableRetry *bool `cfg:"disable_retry" json:"disable_retry,omitempty"`

	// RetryMax is the maximum number of retry attempts. Default is 4.
	RetryMax int `cfg:"retry_max" json:"retry_max,omitempty"`

	// RetryWaitMin is the minimum wait time between retries.
	RetryWaitMin time.Duration `cfg:"retry_wait_min" json:"retry_wait_min,omitempty"`

	// RetryWaitMax is the maximum wait time between retries.
	RetryWaitMax time.Duration `cfg:"retry_wait_max" json:"retry_wait_max,omitempty"`

	// RetryTimeout is the per-attempt timeout. Zero means no per-attempt timeout.
	RetryTimeout time.Duration `cfg:"retry_timeout" json:"retry_timeout,omitempty"`

	// Proxy sets the proxy URL for the HTTP transport.
	Proxy string `cfg:"proxy" json:"proxy,omitempty"`

	// HTTP2 enables HTTP/2 support.
	HTTP2 *bool `cfg:"http2" json:"http2,omitempty"`

	// TLS holds TLS certificate configuration.
	TLS *TLSConfig `cfg:"tls" json:"tls,omitempty"`
}

// ToOption converts the Config into a single OptionClientFn.
// Only non-zero/non-nil fields are applied.
func (c Config) ToOption() OptionClientFn {
	return func(o *optionClientValue) {
		if c.BaseURL != "" {
			o.BaseURL = c.BaseURL
		}

		if len(c.Header) > 0 {
			for k, vs := range c.Header {
				for _, v := range vs {
					o.Header.Add(k, v)
				}
			}
		}

		if c.Timeout > 0 {
			o.Timeout = c.Timeout
		}

		if c.EnableBaseURLCheck != nil {
			o.EnableBaseURLCheck = *c.EnableBaseURLCheck
		}

		if c.EnableEnvValues != nil {
			o.EnableEnvValues = *c.EnableEnvValues
		}

		if c.InsecureSkipVerify != nil {
			o.InsecureSkipVerify = *c.InsecureSkipVerify
		}

		if c.DisableRetry != nil {
			o.DisableRetry = *c.DisableRetry
		}

		if c.RetryMax > 0 {
			o.RetryMax = c.RetryMax
		}

		if c.RetryWaitMin > 0 {
			o.RetryWaitMin = c.RetryWaitMin
		}

		if c.RetryWaitMax > 0 {
			o.RetryWaitMax = c.RetryWaitMax
		}

		if c.RetryTimeout > 0 {
			o.RetryTimeout = c.RetryTimeout
		}

		if c.Proxy != "" {
			o.Proxy = c.Proxy
		}

		if c.HTTP2 != nil {
			o.HTTP2 = *c.HTTP2
		}

		if c.TLS != nil {
			tlsCfg, err := c.TLS.Generate()
			if err == nil && tlsCfg != nil {
				o.TLSConfig = tlsCfg
			}
		}
	}
}

// New creates a Client from the Config, with additional options applied after.
func (c *Config) New(opts ...OptionClientFn) (*Client, error) {
	return New(append([]OptionClientFn{c.ToOption()}, opts...)...)
}
