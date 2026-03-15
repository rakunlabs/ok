# 🏹 OK

[![License](https://img.shields.io/github/license/rakunlabs/ok?color=red&style=flat-square)](https://raw.githubusercontent.com/rakunlabs/ok/main/LICENSE)
[![Coverage](https://img.shields.io/sonar/coverage/rakunlabs_ok?logo=sonarcloud&server=https%3A%2F%2Fsonarcloud.io&style=flat-square)](https://sonarcloud.io/summary/overall?id=rakunlabs_ok)
[![GitHub Workflow Status](https://img.shields.io/github/actions/workflow/status/rakunlabs/ok/test.yml?branch=main&logo=github&style=flat-square&label=ci)](https://github.com/rakunlabs/ok/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/rakunlabs/ok?style=flat-square)](https://goreportcard.com/report/github.com/rakunlabs/ok)
[![Go PKG](https://raw.githubusercontent.com/rakunlabs/.github/main/assets/badges/gopkg.svg)](https://pkg.go.dev/github.com/rakunlabs/ok)

HTTP client library for Go with retryable requests, automatic body draining, base URL resolution, and structured logging.

```sh
go get github.com/rakunlabs/ok
```

## Features

- **Retryable HTTP** with exponential backoff and jitter, configurable per-client and per-request
- **Automatic body drain/close** on every request to ensure connection reuse
- **Base URL resolution** transparently applied at the transport layer
- **Default headers** and per-request context headers
- **HTTP/2 support** including h2c (unencrypted HTTP/2)
- **Custom transport chain** with user-provided RoundTripper wrappers
- **Full TLS configuration** with client certs, custom CA, and insecure skip verify
- **Structured logging** via `log/slog` with a pluggable `Logger` interface
- **Environment variable overrides** (opt-in)
- **Config struct** with `cfg` and `json` tags for file/env deserialization
- **Test helpers** (`oktest` package) for unit testing without a real HTTP server

## Quick Start

```go
package main

import (
    "fmt"
    "net/http"

    "github.com/rakunlabs/ok"
)

func main() {
    client, err := ok.New(
        ok.WithBaseURL("https://api.example.com"),
    )
    if err != nil {
        panic(err)
    }

    req, _ := http.NewRequest(http.MethodGet, "/users", nil)

    var users []struct {
        Name string `json:"name"`
    }

    if err := client.Do(req, ok.ResponseFuncJSON(&users)); err != nil {
        panic(err)
    }

    fmt.Println(users)
}
```

## Usage

### Creating a Client

Use `ok.New` with functional options:

```go
client, err := ok.New(
    ok.WithBaseURL("https://api.example.com/v1"),
    ok.WithTimeout(30 * time.Second),
    ok.WithRetryMax(3),
    ok.WithHeaderSet("Authorization", "Bearer token"),
)
```

### Making Requests

The `Do` method executes a request and passes the response to a callback. The response body is **always drained and closed** after the callback returns, so you never need to close it yourself:

```go
req, _ := http.NewRequest(http.MethodGet, "/resource", nil)

err := client.Do(req, func(resp *http.Response) error {
    // Process the response here.
    // Body is automatically drained and closed after this function returns.
    fmt.Println(resp.StatusCode)
    return nil
})
```

### JSON Responses

`ResponseFuncJSON` checks for a 2xx status code and decodes the body:

```go
var result MyStruct
err := client.Do(req, ok.ResponseFuncJSON(&result))
```

Pass `nil` to only validate the status code without decoding:

```go
err := client.Do(req, ok.ResponseFuncJSON(nil))
```

### Package-Level Do

If you have a plain `*http.Client`, use the package-level `Do`:

```go
httpClient := &http.Client{}
err := ok.Do(httpClient, req, ok.ResponseFuncJSON(&result))
```

## Config Struct

The `Config` struct can be populated from configuration files or environment and converted to options. Fields use `*bool` for optional booleans to distinguish "not set" from `false`.

```go
cfg := &ok.Config{
    BaseURL:  "https://api.example.com",
    Timeout:  30 * time.Second,
    RetryMax: 3,
}

client, err := cfg.New(
    ok.WithHeaderSet("X-Custom", "value"), // additional options
)
```

Or convert to an option for composition:

```go
client, err := ok.New(
    cfg.ToOption(),
    ok.WithLogger(ok.NoopLogger{}),
)
```

<details>
<summary>Config fields</summary>

| Field                | Type                  | Tag                           | Description                             |
|----------------------|-----------------------|-------------------------------|-----------------------------------------|
| `BaseURL`            | `string`              | `cfg:"base_url"`              | Base URL for all requests               |
| `Header`             | `map[string][]string` | `cfg:"header"`                | Default headers                         |
| `Timeout`            | `time.Duration`       | `cfg:"timeout"`               | Overall client timeout                  |
| `EnableBaseURLCheck` | `*bool`               | `cfg:"enable_base_url_check"` | Validate base URL has scheme and host   |
| `EnableEnvValues`    | `*bool`               | `cfg:"enable_env_values"`     | Enable environment variable reading     |
| `InsecureSkipVerify` | `*bool`               | `cfg:"insecure_skip_verify"`  | Skip TLS certificate verification       |
| `DisableRetry`       | `*bool`               | `cfg:"disable_retry"`         | Disable automatic retry                 |
| `RetryMax`           | `int`                 | `cfg:"retry_max"`             | Max retry attempts (default: 4)         |
| `RetryWaitMin`       | `time.Duration`       | `cfg:"retry_wait_min"`        | Min wait between retries (default: 1s)  |
| `RetryWaitMax`       | `time.Duration`       | `cfg:"retry_wait_max"`        | Max wait between retries (default: 30s) |
| `RetryTimeout`       | `time.Duration`       | `cfg:"retry_timeout"`         | Per-attempt timeout (0 = disabled)      |
| `Proxy`              | `string`              | `cfg:"proxy"`                 | Proxy URL (ignored with HTTP/2)         |
| `HTTP2`              | `*bool`               | `cfg:"http2"`                 | Enable HTTP/2 including h2c             |
| `TLS`                | `*TLSConfig`          | `cfg:"tls"`                   | TLS certificate paths                   |

</details>

## Retry

Retry is **enabled by default** with exponential backoff and jitter. The retry transport buffers the request body and re-sends it on each attempt.

| Setting      | Default                              |
|--------------|--------------------------------------|
| Max retries  | 4                                    |
| Min wait     | 1s                                   |
| Max wait     | 30s                                  |
| Backoff      | Exponential with full jitter         |
| Retry policy | 5xx, 429, timeout, connection errors |

### Disable Retry

```go
client, err := ok.New(
    ok.WithDisableRetry(true),
)
```

### Custom Retry Policy

```go
client, err := ok.New(
    ok.WithRetryPolicy(func(ctx context.Context, resp *http.Response, err error) (bool, error) {
        if resp != nil && resp.StatusCode == http.StatusConflict {
            return true, nil // retry on 409
        }
        return ok.DefaultRetryPolicy(ctx, resp, err)
    }),
)
```

### Per-Request Retry Override

Override retry behavior for a single request using context:

```go
ctx := ok.CtxWithRetryPolicy(context.Background(),
    ok.OptionRetry.WithRetryDisable(),
)
req, _ := http.NewRequestWithContext(ctx, http.MethodPost, "/resource", body)
```

Force retry on specific status codes:

```go
ctx := ok.CtxWithRetryPolicy(context.Background(),
    ok.OptionRetry.WithRetryEnabledStatusCodes(http.StatusConflict),
    ok.OptionRetry.WithRetryDisabledStatusCodes(http.StatusTooManyRequests),
)
```

### Per-Attempt Timeout

Set a timeout for each individual attempt (distinct from the overall client timeout):

```go
client, err := ok.New(
    ok.WithTimeout(2 * time.Minute),         // overall timeout
    ok.WithRetryTimeout(10 * time.Second),    // per-attempt timeout
)
```

> Per-attempt timeout is skipped when HTTP/2 is enabled.

## HTTP/2

Enable HTTP/2 including h2c (unencrypted HTTP/2):

```go
client, err := ok.New(
    ok.WithHTTP2(true),
    ok.WithBaseURL("http://localhost:8080"),
)
```

When HTTP/2 is enabled:
- Proxy settings are ignored
- Per-attempt retry timeout is skipped

## TLS

### Insecure Skip Verify

```go
client, err := ok.New(
    ok.WithInsecureSkipVerify(true),
)
```

### Client Certificates and Custom CA

Using the `Config` struct:

```go
cfg := &ok.Config{
    TLS: &ok.TLSConfig{
        CertFile: "/path/to/client.pem",
        KeyFile:  "/path/to/client-key.pem",
        CAFile:   "/path/to/ca.pem",
    },
}
client, err := cfg.New()
```

Or generate a `*tls.Config` directly:

```go
tlsCfg := ok.TLSConfig{
    CertFile: "client.pem",
    KeyFile:  "client-key.pem",
    CAFile:   "ca.pem",
}
cfg, err := tlsCfg.Generate()
if err != nil {
    panic(err)
}

client, err := ok.New(ok.WithTLSConfig(cfg))
```

## Headers

### Default Headers

Applied to every request (only if the request doesn't already set that header):

```go
client, err := ok.New(
    ok.WithHeaderSet("Authorization", "Bearer token"),
    ok.WithHeaderSet("Accept", "application/json"),
    ok.WithUserAgent("myapp/1.0"),
)
```

### Per-Request Headers via Context

```go
header := http.Header{}
header.Set("X-Trace-Id", "abc-123")

ctx := ok.CtxWithHeader(context.Background(), header)
req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "/resource", nil)
```

### Inject Function

For tracing propagation or other per-request modifications:

```go
client, err := ok.New(
    ok.WithInject(func(ctx context.Context, req *http.Request) {
        // e.g., OpenTelemetry propagation
        propagator.Inject(ctx, propagation.HeaderCarrier(req.Header))
    }),
)
```

## Custom Transport

Add RoundTripper wrappers to the transport chain:

```go
client, err := ok.New(
    ok.WithRoundTripper(func(ctx context.Context, rt http.RoundTripper) (http.RoundTripper, error) {
        return myCustomTransport{Base: rt}, nil
    }),
)
```

Or provide a base transport:

```go
client, err := ok.New(
    ok.WithBaseTransport(myTransport),
)
```

### Transport Chain

The transport chain is built innermost to outermost:

```
Base Transport (*http.Transport)
  -> Per-Attempt Timeout Transport
    -> Retry Transport
      -> TransportOK (base URL, headers, inject)
        -> User RoundTripper Wrappers
```

## Logging

Default logger is `slog.Default()`. All retry attempts are logged at `WARN` level.

### Custom Logger

Any type implementing the `Logger` interface works:

```go
type Logger interface {
    Error(msg string, keysAndValues ...any)
    Warn(msg string, keysAndValues ...any)
    Info(msg string, keysAndValues ...any)
    Debug(msg string, keysAndValues ...any)
}
```

Since `*slog.Logger` satisfies this interface, you can pass any slog logger directly:

```go
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

client, err := ok.New(
    ok.WithLogger(logger),
)
```

### Disable Logging

```go
client, err := ok.New(
    ok.WithLogger(ok.NoopLogger{}),
)
```

### Disable Retry Logging Only

```go
client, err := ok.New(
    ok.WithRetryLog(false),
)
```

## Environment Variables

Environment variable reading is **disabled by default**. Enable it with:

```go
client, err := ok.New(
    ok.WithEnableEnvValues(true),
)
```

Or enable globally:

```go
ok.EnableEnvValues = true
```

| Variable                  | Description                            |
|---------------------------|----------------------------------------|
| `OK_BASE_URL`             | Base URL (only if not explicitly set)  |
| `OK_INSECURE_SKIP_VERIFY` | Skip TLS verification (`true`/`false`) |
| `OK_TIMEOUT`              | Client timeout (e.g., `30s`, `1m`)     |
| `OK_RETRY_DISABLE`        | Disable retry (`true`/`false`)         |

Environment variables have **lower precedence** than options set in code.

## Testing

The `oktest` package provides a fake `http.RoundTripper` for unit testing without a real HTTP server:

```go
import (
    "github.com/rakunlabs/ok"
    "github.com/rakunlabs/ok/oktest"
)

func TestMyAPI(t *testing.T) {
    th := &oktest.TransportHandler{}
    th.SetHandler(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
    })

    client, err := ok.New(
        ok.WithBaseTransport(th),
        ok.WithDisableRetry(true),
    )
    if err != nil {
        t.Fatal(err)
    }

    req, _ := http.NewRequest(http.MethodGet, "/api/status", nil)

    var result map[string]string
    if err := client.Do(req, ok.ResponseFuncJSON(&result)); err != nil {
        t.Fatal(err)
    }

    if result["status"] != "ok" {
        t.Errorf("got %q, want %q", result["status"], "ok")
    }
}
```

The handler can be swapped between test cases:

```go
th.SetHandler(newHandler) // thread-safe
```

## Options Reference

<details>
<summary>All options</summary>

| Option                         | Default              | Description                                  |
|--------------------------------|----------------------|----------------------------------------------|
| `WithBaseURL(url)`             | `""`                 | Base URL for resolving relative request URLs |
| `WithEnableBaseURLCheck(bool)` | `false`              | Validate base URL has scheme and host        |
| `WithHeader(http.Header)`      | empty                | Set default headers (cloned)                 |
| `WithHeaderAdd(key, value)`    | -                    | Add a default header value                   |
| `WithHeaderSet(key, value)`    | -                    | Set a default header (replaces existing)     |
| `WithHeaderDel(key)`           | -                    | Remove a default header                      |
| `WithUserAgent(ua)`            | `""`                 | Set User-Agent header                        |
| `WithHTTPClient(client)`       | `nil`                | Use a custom `*http.Client` as base          |
| `WithBaseTransport(rt)`        | `nil`                | Set the innermost transport                  |
| `WithDisableRetry(bool)`       | `false`              | Disable automatic retry                      |
| `WithRetryMax(n)`              | `4`                  | Maximum retry attempts                       |
| `WithRetryWaitMin(d)`          | `1s`                 | Minimum backoff wait                         |
| `WithRetryWaitMax(d)`          | `30s`                | Maximum backoff wait                         |
| `WithRetryTimeout(d)`          | `0`                  | Per-attempt timeout                          |
| `WithRetryPolicy(fn)`          | `DefaultRetryPolicy` | Custom retry policy                          |
| `WithBackoff(fn)`              | `DefaultBackoff`     | Custom backoff function                      |
| `WithRetryLog(bool)`           | `true`               | Log retry attempts                           |
| `WithTimeout(d)`               | `0`                  | Overall client timeout                       |
| `WithInsecureSkipVerify(bool)` | `false`              | Skip TLS verification                        |
| `WithTLSConfig(cfg)`           | `nil`                | Custom `*tls.Config`                         |
| `WithRoundTripper(fn)`         | `nil`                | Add transport wrapper                        |
| `WithProxy(url)`               | `""`                 | Proxy URL                                    |
| `WithHTTP2(bool)`              | `false`              | Enable HTTP/2 + h2c                          |
| `WithInject(fn)`               | `nil`                | Pre-request hook                             |
| `WithLogger(logger)`           | `slog.Default()`     | Set logger                                   |
| `WithEnableEnvValues(bool)`    | `false`              | Enable env var reading                       |

</details>
