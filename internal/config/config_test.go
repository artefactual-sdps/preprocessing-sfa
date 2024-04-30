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
	tmpDir := fs.NewDir(
		t, "",
		fs.WithFile(
			"preprocessing.toml",
			testConfig,
		),
	)
	configFile := tmpDir.Join("preprocessing.toml")

	var c config.Configuration
	found, configFileUsed, err := config.Read(&c, configFile)

	assert.NilError(t, err)
	assert.Equal(t, found, true)
	assert.Equal(t, configFileUsed, configFile)

	assert.Equal(t, c.Debug, true)
	assert.Equal(t, c.Verbosity, 2)
	assert.Equal(t, c.SharedPath, "/home/preprocessing/shared")

	assert.Equal(t, c.Temporal.Address, "host:port")
	assert.Equal(t, c.Temporal.Namespace, "default")
	assert.Equal(t, c.Temporal.TaskQueue, "preprocessing")
	assert.Equal(t, c.Temporal.WorkflowName, "preprocessing")

	assert.Equal(t, c.Worker.MaxConcurrentSessions, 1)
}
