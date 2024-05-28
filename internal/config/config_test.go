package config_test

import (
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/config"
)

const testConfig = `# Config
debug = true
verbosity = 2
sharedPath = "/home/preprocessing/shared"
[temporal]
address = "host:port"
namespace = "default"
taskQueue = "preprocessing"
workflowName = "preprocessing"
[worker]
maxConcurrentSessions = 1
`

func TestConfig(t *testing.T) {
	t.Parallel()

	type test struct {
		name       string
		configFile string
		toml       string
		wantFound  bool
		wantCfg    config.Configuration
		wantErr    string
	}

	for _, tc := range []test{
		{
			name:       "Loads configuration from a TOML file",
			configFile: "preprocessing.toml",
			toml:       testConfig,
			wantFound:  true,
			wantCfg: config.Configuration{
				Debug:      true,
				Verbosity:  2,
				SharedPath: "/home/preprocessing/shared",
				Temporal: config.Temporal{
					Address:      "host:port",
					Namespace:    "default",
					TaskQueue:    "preprocessing",
					WorkflowName: "preprocessing",
				},
				Worker: config.WorkerConfig{
					MaxConcurrentSessions: 1,
				},
			},
		},
		{
			name:       "Errors when configuration values are not valid",
			configFile: "preprocessing.toml",
			wantFound:  true,
			wantErr: `invalid configuration:
SharedPath: missing required value
TaskQueue: missing required value
WorkflowName: missing required value`,
		},
		{
			name:       "Errors when MaxConcurrentSessions is less than 1",
			configFile: "preprocessing.toml",
			toml: `# Config
sharedPath = "/home/preprocessing/shared"
[temporal]
taskQueue = "preprocessing"
workflowName = "preprocessing"
[worker]
maxConcurrentSessions = -1
`,
			wantFound: true,
			wantErr:   `Worker.MaxConcurrentSessions: -1 is less than the minimum value (1)`,
		},
		{
			name:       "Errors when TOML is invalid",
			configFile: "preprocessing.toml",
			toml:       "bad TOML",
			wantFound:  true,
			wantErr:    "failed to read configuration file: While parsing config: toml: expected character =",
		},
		{
			name:      "Errors when no config file is found in the default paths",
			wantFound: false,
			wantErr:   "Config File \"preprocessing\" Not Found in",
		},
		{
			name:       "Errors when the given configFile is not found",
			configFile: "missing.toml",
			wantFound:  false,
			wantErr:    "configuration file not found: ",
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tmpDir := fs.NewDir(t, "preprocessing-test", fs.WithFile("preprocessing.toml", tc.toml))

			configFile := ""
			if tc.configFile != "" {
				configFile = tmpDir.Join(tc.configFile)
			}

			var c config.Configuration
			found, configFileUsed, err := config.Read(&c, configFile)
			if tc.wantErr != "" {
				assert.Equal(t, found, tc.wantFound)
				assert.ErrorContains(t, err, tc.wantErr)
				return
			}

			assert.NilError(t, err)
			assert.Equal(t, found, true)
			assert.Equal(t, configFileUsed, configFile)
			assert.DeepEqual(t, c, tc.wantCfg)
		})
	}
}
