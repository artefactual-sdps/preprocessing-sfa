package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type ConfigurationValidator interface {
	Validate() error
}

type Configuration struct {
	Verbosity  int
	Debug      bool
	SharedPath string
	Temporal   Temporal
	Worker     WorkerConfig
}

type Temporal struct {
	Address      string
	Namespace    string
	TaskQueue    string
	WorkflowName string
}

type WorkerConfig struct {
	MaxConcurrentSessions int
}

func (c Configuration) Validate() error { return nil }

func Read(config *Configuration, configFile string) (found bool, configFileUsed string, err error) {
	v := viper.New()

	v.AddConfigPath(".")
	v.AddConfigPath("$HOME/.config/")
	v.AddConfigPath("/etc")
	v.SetConfigName("preprocessing")
	v.SetEnvPrefix("ENDURO_PREPROCESSING")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Defaults.
	v.SetDefault("Worker.MaxConcurrentSessions", 1)

	if configFile != "" {
		v.SetConfigFile(configFile)
	}

	err = v.ReadInConfig()
	_, ok := err.(viper.ConfigFileNotFoundError)
	if !ok {
		found = true
	}
	if found && err != nil {
		return found, configFileUsed, fmt.Errorf("failed to read configuration file: %w", err)
	}

	err = v.Unmarshal(config)
	if err != nil {
		return found, configFileUsed, fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	if err := config.Validate(); err != nil {
		return found, configFileUsed, fmt.Errorf("failed to validate the provided config: %w", err)
	}

	configFileUsed = v.ConfigFileUsed()

	return found, configFileUsed, nil
}
