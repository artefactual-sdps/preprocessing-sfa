package activities_test

import (
	"fmt"
	"os"
	"testing"

	temporalsdk_activity "go.temporal.io/sdk/activity"
	temporalsdk_testsuite "go.temporal.io/sdk/testsuite"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/activities"
)

func TestAddPREMISObjects(t *testing.T) {
	t.Parallel()

	// Normally populated files (for execution expected to work).
	contentFilesNormal := fs.NewDir(t, "",
		fs.WithDir("digitized_Vecteur_SIP",
			fs.WithDir("header",
				fs.WithDir("xsd",
					fs.WithFile("arelda.xsd", ""),
				),
				fs.WithFile("metadata.xml", sipManifest),
			),
			fs.WithDir("content",
				fs.WithDir("d_0000001",
					fs.WithFile("00000001.jp2", ""),
					fs.WithFile("00000001_PREMIS.xml", ""),
					fs.WithFile("00000002.jp2", ""),
					fs.WithFile("00000002_PREMIS.xml", ""),
					fs.WithFile("Prozess_Digitalisierung_PREMIS.xml", ""),
				),
			),
		),
	)

	premisFilePathNormal := contentFilesNormal.Join("digitized_Vecteur_SIP", "metadata", "premis.xml")

	// No files (for execution expected to work).
	contentNoFiles := fs.NewDir(t, "",
		fs.WithDir("digitized_Vecteur_SIP",
			fs.WithDir("content",
				fs.WithDir("content",
					fs.WithDir("d_0000001"),
				),
			),
		),
	)

	premisFilePathNoFiles := contentNoFiles.Join("digitized_Vecteur_SIP", "metadata", "premis.xml")

	tests := []struct {
		name       string
		params     activities.AddPREMISObjectsParams
		result     activities.AddPREMISObjectsResult
		wantPREMIS string
		wantErr    string
	}{
		{
			name: "Add PREMIS objects for normal content",
			params: activities.AddPREMISObjectsParams{
				SIP:            testSIP(t, contentFilesNormal.Join("digitized_Vecteur_SIP")),
				PREMISFilePath: premisFilePathNormal,
			},
			result: activities.AddPREMISObjectsResult{},
		},
		{
			name: "Error when manifest is missing",
			params: activities.AddPREMISObjectsParams{
				SIP:            testSIP(t, contentNoFiles.Join("digitized_Vecteur_SIP")),
				PREMISFilePath: premisFilePathNoFiles,
			},
			result: activities.AddPREMISObjectsResult{},
			wantErr: fmt.Sprintf(
				"open manifest file: open %s: no such file or directory",
				contentNoFiles.Join("digitized_Vecteur_SIP", "header", "metadata.xml"),
			),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ts := &temporalsdk_testsuite.WorkflowTestSuite{}
			env := ts.NewTestActivityEnvironment()
			env.RegisterActivityWithOptions(
				activities.NewAddPREMISObjects().Execute,
				temporalsdk_activity.RegisterOptions{Name: activities.AddPREMISObjectsName},
			)

			var res activities.AddPREMISObjectsResult
			future, err := env.ExecuteActivity(activities.AddPREMISObjectsName, tt.params)

			if tt.wantErr != "" {
				if err == nil {
					t.Errorf("error is nil, expecting: %q", tt.wantErr)
				} else {
					assert.ErrorContains(t, err, tt.wantErr)
				}

				return
			}
			assert.NilError(t, err)

			future.Get(&res)
			assert.DeepEqual(t, res, tt.result)

			b, err := os.ReadFile(tt.params.PREMISFilePath)
			assert.NilError(t, err)
			assert.Equal(t, string(b), `<?xml version="1.0" encoding="UTF-8"?>
<premis:premis xmlns:premis="http://www.loc.gov/premis/v3" xmlns:xlink="http://www.w3.org/1999/xlink" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.loc.gov/premis/v3 https://www.loc.gov/standards/premis/premis.xsd" version="3.0">
  <premis:object xsi:type="premis:file">
    <premis:objectIdentifier>
      <premis:objectIdentifierType>local</premis:objectIdentifierType>
      <premis:objectIdentifierValue>_zodSTSD0nv05CpOp6JoV3X</premis:objectIdentifierValue>
    </premis:objectIdentifier>
    <premis:objectCharacteristics>
      <premis:format>
        <premis:formatDesignation>
          <premis:formatName/>
        </premis:formatDesignation>
      </premis:format>
    </premis:objectCharacteristics>
    <premis:originalName>data/objects/digitized_Vecteur_SIP/content/d_0000001/00000001.jp2</premis:originalName>
  </premis:object>
  <premis:object xsi:type="premis:file">
    <premis:objectIdentifier>
      <premis:objectIdentifierType>local</premis:objectIdentifierType>
      <premis:objectIdentifierValue>_WuDmXAs5UDwKTGVLsCcZxa</premis:objectIdentifierValue>
    </premis:objectIdentifier>
    <premis:objectCharacteristics>
      <premis:format>
        <premis:formatDesignation>
          <premis:formatName/>
        </premis:formatDesignation>
      </premis:format>
    </premis:objectCharacteristics>
    <premis:originalName>data/objects/digitized_Vecteur_SIP/content/d_0000001/00000001_PREMIS.xml</premis:originalName>
  </premis:object>
  <premis:object xsi:type="premis:file">
    <premis:objectIdentifier>
      <premis:objectIdentifierType>local</premis:objectIdentifierType>
      <premis:objectIdentifierValue>_rlPKJX9ZcAl4ooc4IfoIkM</premis:objectIdentifierValue>
    </premis:objectIdentifier>
    <premis:objectCharacteristics>
      <premis:format>
        <premis:formatDesignation>
          <premis:formatName/>
        </premis:formatDesignation>
      </premis:format>
    </premis:objectCharacteristics>
    <premis:originalName>data/objects/digitized_Vecteur_SIP/content/d_0000001/00000002.jp2</premis:originalName>
  </premis:object>
  <premis:object xsi:type="premis:file">
    <premis:objectIdentifier>
      <premis:objectIdentifierType>local</premis:objectIdentifierType>
      <premis:objectIdentifierValue>_Ohk77y2DJa82RXqsWG4S90</premis:objectIdentifierValue>
    </premis:objectIdentifier>
    <premis:objectCharacteristics>
      <premis:format>
        <premis:formatDesignation>
          <premis:formatName/>
        </premis:formatDesignation>
      </premis:format>
    </premis:objectCharacteristics>
    <premis:originalName>data/objects/digitized_Vecteur_SIP/content/d_0000001/00000002_PREMIS.xml</premis:originalName>
  </premis:object>
  <premis:object xsi:type="premis:file">
    <premis:objectIdentifier>
      <premis:objectIdentifierType>local</premis:objectIdentifierType>
      <premis:objectIdentifierValue>_cQ6sm5CChWVqtqmrWvne0W</premis:objectIdentifierValue>
    </premis:objectIdentifier>
    <premis:objectCharacteristics>
      <premis:format>
        <premis:formatDesignation>
          <premis:formatName/>
        </premis:formatDesignation>
      </premis:format>
    </premis:objectCharacteristics>
    <premis:originalName>data/metadata/Prozess_Digitalisierung_PREMIS_d_0000001.xml</premis:originalName>
  </premis:object>
</premis:premis>
`)
		})
	}
}
