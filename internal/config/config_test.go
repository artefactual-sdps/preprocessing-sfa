package config_test

import (
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/config"
)

const testConfig = `# Config
debug = true
[temporal]
address = "host:port"
`

func TestConfig(t *testing.T) {
	tmpDir := fs.NewDir(
		t, "",
		fs.WithFile(
			"preprocessing_sfa.toml",
			testConfig,
		),
	)
	configFile := tmpDir.Join("preprocessing_sfa.toml")

	var c config.Configuration
	found, configFileUsed, err := config.Read(&c, configFile)

	assert.NilError(t, err)
	assert.Equal(t, found, true)
	assert.Equal(t, configFileUsed, configFile)
	assert.Equal(t, c.Temporal.Address, "host:port")
}
