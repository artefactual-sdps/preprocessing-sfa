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
			name:   "disabled config",
			config: Config{Enabled: false},
		},
		{
			name: "valid config",
			config: Config{
				Enabled:      true,
				URL:          "http://apis.example.test",
				Timeout:      DefaultTimeout,
				PollInterval: DefaultPollInterval,
				Token:        "mock-token",
			},
		},
		{
			name: "missing URL",
			config: Config{
				Enabled:      true,
				Timeout:      DefaultTimeout,
				PollInterval: DefaultPollInterval,
			},
			wantErr: "APIS.URL: missing required value",
		},
		{
			name: "negative timeout",
			config: Config{
				Enabled:      true,
				URL:          "http://apis.example.test",
				Timeout:      -1 * time.Second,
				PollInterval: DefaultPollInterval,
			},
			wantErr: "APIS.Timeout: value -1s is less than 0",
		},
		{
			name: "zero poll interval",
			config: Config{
				Enabled:      true,
				URL:          "http://apis.example.test",
				Timeout:      DefaultTimeout,
				PollInterval: 0,
			},
			wantErr: "APIS.PollInterval: value 0s is less than or equal to 0",
		},
		{
			name: "negative poll interval",
			config: Config{
				Enabled:      true,
				URL:          "http://apis.example.test",
				Timeout:      DefaultTimeout,
				PollInterval: -1 * time.Second,
			},
			wantErr: "APIS.PollInterval: value -1s is less than or equal to 0",
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
