package apis

import (
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

func TestConfigValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  Config
		wantErr string
	}{
		{
			name: "valid config",
			config: Config{
				URL:          "http://apis.example.test",
				Timeout:      DefaultTimeout,
				PollInterval: DefaultPollInterval,
				Token:        "mock-token",
			},
		},
		{
			name: "missing URL",
			config: Config{
				Timeout:      DefaultTimeout,
				PollInterval: DefaultPollInterval,
			},
			wantErr: "missing APIS URL",
		},
		{
			name: "negative timeout",
			config: Config{
				URL:          "http://apis.example.test",
				Timeout:      -1 * time.Second,
				PollInterval: DefaultPollInterval,
			},
			wantErr: "invalid APIS timeout: -1s",
		},
		{
			name: "zero poll interval",
			config: Config{
				URL:          "http://apis.example.test",
				Timeout:      DefaultTimeout,
				PollInterval: 0,
			},
			wantErr: "invalid APIS poll interval: 0s",
		},
		{
			name: "negative poll interval",
			config: Config{
				URL:          "http://apis.example.test",
				Timeout:      DefaultTimeout,
				PollInterval: -1 * time.Second,
			},
			wantErr: "invalid APIS poll interval: -1s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.config.Validate()
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}

			assert.NilError(t, err)
		})
	}
}
