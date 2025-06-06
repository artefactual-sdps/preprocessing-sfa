package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/artefactual-sdps/temporal-activities/bagcreate"
	"github.com/artefactual-sdps/temporal-activities/ffvalidate"
	"github.com/spf13/viper"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/ais"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/fvalidate"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/persistence"
)

type ConfigurationValidator interface {
	Validate() error
}

type Configuration struct {
	// Debug toggles human readable logs or JSON logs (default).
	Debug bool

	// Verbosity sets the verbosity level of log messages, with 0 (default)
	// logging only critical messages and each higher number increasing the
	// number of messages logged.
	Verbosity int

	// SharedPath is a file path that both Preprocessing and Enduro can access
	// (required).
	//
	// Enduro will deposit transfers in SharedPath for preprocessing.
	// Preprocessing must write transfer updates to SharedPath for retrieval by
	// Enduro and preservation processing.
	SharedPath string

	// CheckDuplicates enables or disables a check for SIPs that have already
	// been processed. When enabled, the persistence configuration below will
	// be required, and a SIP that has already been processed will fail the
	// preprocessing workflow.
	CheckDuplicates bool

	Persistence  persistence.Config
	Temporal     Temporal
	Worker       WorkerConfig
	Bagit        bagcreate.Config
	AIS          ais.Config
	FileFormat   ffvalidate.Config
	FileValidate fvalidate.Config
}

type Temporal struct {
	// Address is the Temporal server host and port (default: "localhost:7233").
	Address string

	// Namespace is the Temporal namespace the preprocessing worker should run
	// in (default: "default").
	Namespace string

	// TaskQueue is the Temporal task queue from which the preprocessing worker
	// will pull tasks (required).
	TaskQueue string

	// WorkflowName is the name of the preprocessing Temporal workflow
	// (required).
	WorkflowName string
}

type WorkerConfig struct {
	// MaxConcurrentSessions limits the number of workflow sessions the
	// preprocessing worker can handle simultaneously (default: 1).
	MaxConcurrentSessions int
}

func (c Configuration) Validate() error {
	var errs error

	// Verify that the required fields have values.
	if c.SharedPath == "" {
		errs = errors.Join(errs, errRequired("SharedPath"))
	}
	if c.Temporal.TaskQueue == "" {
		errs = errors.Join(errs, errRequired("Temporal.TaskQueue"))
	}
	if c.Temporal.WorkflowName == "" {
		errs = errors.Join(errs, errRequired("Temporal.WorkflowName"))
	}

	// Verify that MaxConcurrentSessions is >= 1.
	if c.Worker.MaxConcurrentSessions < 1 {
		errs = errors.Join(errs, fmt.Errorf(
			"Worker.MaxConcurrentSessions: %d is less than the minimum value (1)",
			c.Worker.MaxConcurrentSessions,
		))
	}

	if err := c.Bagit.Validate(); err != nil {
		errs = errors.Join(errs, fmt.Errorf("Bagit.%v", err))
	}

	if c.CheckDuplicates {
		if c.Persistence.DSN == "" {
			errs = errors.Join(errs, errRequired("Persistence.DSN"))
		}
		if c.Persistence.Driver == "" {
			errs = errors.Join(errs, errRequired("Persistence.Driver"))
		}
	}

	return errs
}

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
		// Viper will not return a viper.ConfigFileNotFoundError error when
		// SetConfigFile() is passed a path to a file that doesn't exist, so we
		// need to check ourselves.
		if _, err := os.Stat(configFile); errors.Is(err, os.ErrNotExist) {
			return false, "", fmt.Errorf("configuration file not found: %s", configFile)
		}

		v.SetConfigFile(configFile)
	}

	if err = v.ReadInConfig(); err != nil {
		switch err.(type) {
		case viper.ConfigFileNotFoundError:
			return false, "", err
		default:
			return true, "", fmt.Errorf("failed to read configuration file: %w", err)
		}
	}

	err = v.Unmarshal(config)
	if err != nil {
		return true, "", fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	if err := config.Validate(); err != nil {
		return true, "", errors.Join(errors.New("invalid configuration"), err)
	}

	return true, v.ConfigFileUsed(), nil
}

func errRequired(name string) error {
	return fmt.Errorf("%s: missing required value", name)
}
