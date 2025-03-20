package fvalidate_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"gotest.tools/v3/assert"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/fvalidate"
)

func TestFormatIDs(t *testing.T) {
	t.Parallel()

	v := fvalidate.NewVeraPDFValidator("", fvalidate.RunCommand, logr.Discard())
	got := v.FormatIDs()

	assert.DeepEqual(t, got, []string{
		"fmt/95",   // PDF/A 1a
		"fmt/354",  // PDF/A 1b
		"fmt/476",  // PDF/A 2a
		"fmt/477",  // PDF/A 2b
		"fmt/478",  // PDF/A 2u
		"fmt/479",  // PDF/A 3a
		"fmt/480",  // PDF/A 3b
		"fmt/481",  // PDF/A 3u
		"fmt/1910", // PDF/A 4
		"fmt/1911", // PDF/A 4e
		"fmt/1912", // PDF/A 4f
	})
}

func TestName(t *testing.T) {
	t.Parallel()

	v := fvalidate.NewVeraPDFValidator("", fvalidate.RunCommand, logr.Discard())
	got := v.Name()

	assert.Equal(t, got, "veraPDF")
}

func empty(td string) string {
	return ""
}

func TestValidate(t *testing.T) {
	t.Parallel()

	type test struct {
		name    string
		cmd     string
		path    func(td string) string
		want    func(td string) string
		wantErr func(td string) string
	}
	for _, tt := range []test{
		{
			name: "Does nothing when cmd is not set",
			path: empty,
		},
		{
			name: "Errors when path doesn't exist",
			cmd:  "echo",
			path: func(td string) string { return td + "/foo" },
			wantErr: func(td string) string {
				return fmt.Sprintf("validate: file not found: %s/foo", td)
			},
		},
		{
			name: "Returns nothing when no error",
			cmd:  "echo",
			path: func(td string) string { return td },
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			v := fvalidate.NewVeraPDFValidator(tt.cmd, fvalidate.RunCommand, logr.Discard())
			td := t.TempDir()
			got, err := v.Validate(tt.path(td))
			if tt.wantErr != nil {
				assert.Error(t, err, tt.wantErr(td))
				return
			}

			assert.NilError(t, err)

			if tt.want != nil {
				assert.Equal(t, got, tt.want(td))
			} else {
				assert.Equal(t, got, "")
			}
		})
	}
}

func TestVeraPDFVersion(t *testing.T) {
	runFunc := func(string, ...string) (string, error) {
		return "veraPDF 1.1\nSome more text", nil
	}

	v := fvalidate.NewVeraPDFValidator("veraPDF", runFunc, logr.Discard())
	version, err := v.Version()

	assert.NilError(t, err)
	assert.Assert(t, strings.Contains(version, "veraPDF 1.1"))
}

func TestVeraPDFVersionError(t *testing.T) {
	runFunc := func(string, ...string) (string, error) {
		return "", fmt.Errorf("exit status 1")
	}

	v := fvalidate.NewVeraPDFValidator("badcommand", runFunc, logr.Discard())
	version, err := v.Version()

	assert.Error(t, err, "exit status 1")
	assert.Equal(t, version, "")
}
