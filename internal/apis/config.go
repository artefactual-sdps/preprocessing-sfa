package apis

import (
	"fmt"
	"time"
)

const (
	DefaultTimeout      = 10 * time.Second
	DefaultPollInterval = 30 * time.Second
)

type Config struct {
	// URL is the APIS base URL.
	URL string
	// Timeout configures APIS HTTP client timeout.
	Timeout time.Duration
	// PollInterval configures the interval between APIS import task status polls.
	PollInterval time.Duration
	// Token overrides the token provider and is mainly useful for local or mock
	// APIS deployments that expect a fixed bearer token.
	Token string
	// TODO: Add OIDC config for gotools/clientauth token provider.
}

func (c Config) Validate() error {
	if c.URL == "" {
		return fmt.Errorf("missing APIS URL")
	}
	if c.Timeout < 0 {
		return fmt.Errorf("invalid APIS timeout: %s", c.Timeout)
	}
	if c.PollInterval <= 0 {
		return fmt.Errorf("invalid APIS poll interval: %s", c.PollInterval)
	}
	return nil
}
