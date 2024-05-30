package activities_test

import (
	"fmt"
	"testing"

	temporalsdk_activity "go.temporal.io/sdk/activity"
	temporalsdk_testsuite "go.temporal.io/sdk/testsuite"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/activities"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/enums"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/sip"
)

func TestValidateStructure(t *testing.T) {
	t.Parallel()

	vecteurAIP, err := sip.NewSIP(fs.NewDir(t, "",
		fs.WithDir("additional",
			fs.WithFile("UpdatedAreldaMetadata.xml", ""),
		),
		fs.WithDir("content",
			fs.WithDir("content",
				fs.WithDir("d_0000001"),
			),
			fs.WithDir("header",
				fs.WithDir("xsd",
					fs.WithFile("arelda.xsd", ""),
				),
			),
		),
	).Path())
	assert.NilError(t, err)

	vecteurSIP, err := sip.NewSIP(fs.NewDir(t, "",
		fs.WithDir("content",
			fs.WithDir("d_0000001"),
		),
		fs.WithDir("header",
			fs.WithFile("metadata.xml", ""),
			fs.WithDir("xsd",
				fs.WithFile("arelda.xsd", ""),
			),
		),
	).Path())
	assert.NilError(t, err)

	unexpectedPiecesSIP, err := sip.NewSIP(fs.NewDir(t, "",
		fs.WithDir("content",
			fs.WithDir("d_0000001"),
			fs.WithFile("unexpected.txt", ""),
		),
		fs.WithDir("header",
			fs.WithFile("metadata.xml", ""),
			fs.WithDir("xsd",
				fs.WithFile("arelda.xsd", ""),
			),
		),
		fs.WithDir("unexpected"),
	).Path())
	assert.NilError(t, err)

	tests := []struct {
		name    string
		params  activities.ValidateStructureParams
		wantErr string
	}{
		{
			name:   "Validates a Vecteur AIP",
			params: activities.ValidateStructureParams{SIP: vecteurAIP},
		},
		{
			name:   "Validates a Vecteur SIP",
			params: activities.ValidateStructureParams{SIP: vecteurSIP},
		},
		{
			name:   "Fails to validate a SIP with unexpected components",
			params: activities.ValidateStructureParams{SIP: unexpectedPiecesSIP},
			wantErr: fmt.Sprintf(
				"%s\n%s",
				fmt.Sprintf("unexpected directory: %q", unexpectedPiecesSIP.Path+"/unexpected"),
				fmt.Sprintf("unexpected file: %q", unexpectedPiecesSIP.Path+"/content/unexpected.txt"),
			),
		},
		{
			name:   "Fails to validate a SIP with missing components",
			params: activities.ValidateStructureParams{SIP: sip.SIP{Type: enums.SIPTypeVecteurAIP}},
			wantErr: fmt.Sprintf(
				"%s\n%s\n%s\n%s\n%s",
				"content folder: stat : no such file or directory",
				"metadata file: stat : no such file or directory",
				"XSD file: stat : no such file or directory",
				"read SIP folder: open : no such file or directory",
				"read content folder: open : no such file or directory",
			),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ts := &temporalsdk_testsuite.WorkflowTestSuite{}
			env := ts.NewTestActivityEnvironment()
			env.RegisterActivityWithOptions(
				activities.NewValidateStructure().Execute,
				temporalsdk_activity.RegisterOptions{Name: activities.ValidateStructureName},
			)

			_, err := env.ExecuteActivity(activities.ValidateStructureName, tt.params)

			if tt.wantErr != "" {
				if err == nil {
					t.Errorf("error is nil, expecting: %q", tt.wantErr)
				} else {
					assert.ErrorContains(t, err, tt.wantErr)
				}

				return
			}

			assert.NilError(t, err)
		})
	}
}
