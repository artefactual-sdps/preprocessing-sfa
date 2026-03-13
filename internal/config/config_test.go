package config_test

import (
	"testing"
	"time"

	"github.com/artefactual-sdps/temporal-activities/bagcreate"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/apis"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/config"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/persistence"
)

const testConfig = `# Config
debug = true
verbosity = 2
sharedPath = "/home/preprocessing/shared"
checkDuplicates = true
[persistence]
dsn = "file:/path/to/fake.db"
driver = "sqlite3"
migrate = true
[temporal]
address = "host:port"
namespace = "default"
taskQueue = "preprocessing"
workflowName = "preprocessing"
[worker]
maxConcurrentSessions = 1
[bagit]
checksumAlgorithm = "md5"
[apis]
url = "http://apis.example.test"
`

func TestConfig(t *testing.T) {
	t.Parallel()

	type test struct {
		name            string
		configFile      string
		toml            string
		wantFound       bool
		wantCfg         config.Configuration
		wantErr         string
		wantErrContains string
	}

	for _, tc := range []test{
		{
			name:       "Loads configuration from a TOML file",
			configFile: "preprocessing.toml",
			toml:       testConfig,
			wantFound:  true,
			wantCfg: config.Configuration{
				Debug:           true,
				Verbosity:       2,
				SharedPath:      "/home/preprocessing/shared",
				CheckDuplicates: true,
				Persistence: persistence.Config{
					DSN:     "file:/path/to/fake.db",
					Driver:  "sqlite3",
					Migrate: true,
				},
				Temporal: config.Temporal{
					Address:      "host:port",
					Namespace:    "default",
					TaskQueue:    "preprocessing",
					WorkflowName: "preprocessing",
				},
				Worker: config.WorkerConfig{
					MaxConcurrentSessions: 1,
				},
				Bagit: bagcreate.Config{
					ChecksumAlgorithm: "md5",
				},
				APIS: apis.Config{
					URL:          "http://apis.example.test",
					Timeout:      apis.DefaultTimeout,
					PollInterval: apis.DefaultPollInterval,
				},
			},
		},
		{
			name:       "Errors when configuration values are not valid",
			configFile: "preprocessing.toml",
			wantFound:  true,
			wantErr: `invalid configuration
SharedPath: missing required value
Temporal.TaskQueue: missing required value
Temporal.WorkflowName: missing required value
missing APIS URL`,
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
[apis]
url = "http://apis.example.test"
`,
			wantFound: true,
			wantErr: `invalid configuration
Worker.MaxConcurrentSessions: -1 is less than the minimum value (1)`,
		},
		{
			name:       "Errors when bagit checksumAlgorithm is invalid",
			configFile: "preprocessing.toml",
			toml: `# Config
sharedPath = "/home/preprocessing/shared"
[temporal]
taskQueue = "preprocessing"
workflowName = "preprocessing"
[bagit]
checksumAlgorithm = "unknown"
[apis]
url = "http://apis.example.test"
`,
			wantFound: true,
			wantErr: `invalid configuration
Bagit.ChecksumAlgorithm: invalid value "unknown", must be one of (md5, sha1, sha256, sha512)`,
		},
		{
			name:       "Errors when persistence configuration is missing",
			configFile: "preprocessing.toml",
			toml: `# Config
sharedPath = "/home/preprocessing/shared"
checkDuplicates = true
[temporal]
taskQueue = "preprocessing"
workflowName = "preprocessing"
[apis]
url = "http://apis.example.test"
`,
			wantFound: true,
			wantErr: `invalid configuration
Persistence.DSN: missing required value
Persistence.Driver: missing required value`,
		},
		{
			name:       "Loads APIS defaults when only URL is configured",
			configFile: "preprocessing.toml",
			toml: `# Config
sharedPath = "/home/preprocessing/shared"
[temporal]
taskQueue = "preprocessing"
workflowName = "preprocessing"
[apis]
url = "http://apis.example.test"
`,
			wantFound: true,
			wantCfg: config.Configuration{
				SharedPath: "/home/preprocessing/shared",
				Temporal: config.Temporal{
					TaskQueue:    "preprocessing",
					WorkflowName: "preprocessing",
				},
				Worker: config.WorkerConfig{
					MaxConcurrentSessions: 1,
				},
				APIS: apis.Config{
					URL:          "http://apis.example.test",
					Timeout:      apis.DefaultTimeout,
					PollInterval: apis.DefaultPollInterval,
				},
			},
		},
		{
			name:       "Errors when APIS URL is missing",
			configFile: "preprocessing.toml",
			toml: `# Config
sharedPath = "/home/preprocessing/shared"
[temporal]
taskQueue = "preprocessing"
workflowName = "preprocessing"
`,
			wantFound: true,
			wantErr: `invalid configuration
missing APIS URL`,
		},
		{
			name:       "Errors when APIS timeout is invalid",
			configFile: "preprocessing.toml",
			toml: `# Config
sharedPath = "/home/preprocessing/shared"
[temporal]
taskQueue = "preprocessing"
workflowName = "preprocessing"
[apis]
url = "http://apis.example.test"
timeout = "-1s"
`,
			wantFound: true,
			wantErr: `invalid configuration
invalid APIS timeout: -1s`,
		},
		{
			name:       "Errors when APIS poll interval is invalid",
			configFile: "preprocessing.toml",
			toml: `# Config
sharedPath = "/home/preprocessing/shared"
[temporal]
taskQueue = "preprocessing"
workflowName = "preprocessing"
[apis]
url = "http://apis.example.test"
pollInterval = "-1s"
`,
			wantFound: true,
			wantErr: `invalid configuration
invalid APIS poll interval: -1s`,
		},
		{
			name:       "Loads explicit APIS timeout and poll interval",
			configFile: "preprocessing.toml",
			toml: `# Config
sharedPath = "/home/preprocessing/shared"
[temporal]
taskQueue = "preprocessing"
workflowName = "preprocessing"
[apis]
url = "http://apis.example.test"
timeout = "45s"
pollInterval = "2m"
token = "mock-token"
`,
			wantFound: true,
			wantCfg: config.Configuration{
				SharedPath: "/home/preprocessing/shared",
				Temporal: config.Temporal{
					TaskQueue:    "preprocessing",
					WorkflowName: "preprocessing",
				},
				Worker: config.WorkerConfig{
					MaxConcurrentSessions: 1,
				},
				APIS: apis.Config{
					URL:          "http://apis.example.test",
					Timeout:      45 * time.Second,
					PollInterval: 2 * time.Minute,
					Token:        "mock-token",
				},
			},
		},
		{
			name:       "Errors when TOML is invalid",
			configFile: "preprocessing.toml",
			toml:       "bad TOML",
			wantFound:  true,
			wantErr:    "failed to read configuration file: While parsing config: toml: expected character =",
		},
		{
			name:            "Errors when no config file is found in the default paths",
			wantFound:       false,
			wantErrContains: "Config File \"preprocessing\" Not Found in \"[",
		},
		{
			name:            "Errors when the given configFile is not found",
			configFile:      "missing.toml",
			wantFound:       false,
			wantErrContains: "configuration file not found: ",
		},
	} {
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
				assert.Error(t, err, tc.wantErr)
				return
			}
			if tc.wantErrContains != "" {
				assert.Equal(t, found, tc.wantFound)
				assert.ErrorContains(t, err, tc.wantErrContains)
				return
			}

			assert.NilError(t, err)
			assert.Equal(t, found, true)
			assert.Equal(t, configFileUsed, configFile)
			assert.DeepEqual(t, c, tc.wantCfg)
		})
	}
}
