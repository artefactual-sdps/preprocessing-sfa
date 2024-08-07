package activities_test

import (
	"fmt"
	"path/filepath"
	"testing"

	temporalsdk_activity "go.temporal.io/sdk/activity"
	temporalsdk_testsuite "go.temporal.io/sdk/testsuite"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/activities"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/sip"
)

const manifest = `
<?xml version="1.0" encoding="UTF-8"?>
<paket
	xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
	xmlns:xip="http://www.tessella.com/XIP/v4"
	xmlns="http://bar.admin.ch/arelda/v4"
	xmlns:xs="http://www.w3.org/2001/XMLSchema"
	xmlns:submissionTests="http://bar.admin.ch/submissionTestResult" xsi:type="paketAIP" schemaVersion="5.0">
	<paketTyp>AIP</paketTyp>
	<globaleAIPId>909c56e9-e334-4c0a-9736-f92c732149d9</globaleAIPId>
	<lokaleAIPId>fa5fb285-fa45-44e4-8d85-77ec1d774403</lokaleAIPId>
	<version>1</version>
	<inhaltsverzeichnis>
		<ordner>
			<name>header</name>
			<ordner>
				<name>old</name>
				<ordner>
					<name>SIP</name>
					<datei id="OLD_SIP">
						<name>metadata.xml</name>
						<originalName>metadata.xml</originalName>
						<pruefalgorithmus>MD5</pruefalgorithmus>
						<pruefsumme>43c533d499c572fca699e77e06295ba3</pruefsumme>
					</datei>
				</ordner>
			</ordner>
			<ordner>
				<name>xsd</name>
				<datei id="_xAlSBc3dYcypUMvN8HzeN5">
					<name>arelda.xsd</name>
					<originalName>arelda.xsd</originalName>
					<pruefalgorithmus>MD5</pruefalgorithmus>
					<pruefsumme>f8454632e1ebf97e0aa8d9527ce2641f</pruefsumme>
				</datei>
			</ordner>
		</ordner>
		<ordner>
			<name>content</name>
			<ordner>
				<name>d_0000001</name>
				<datei id="_SRpeVgb4xGImymb23OH1od">
					<name>00000001_PREMIS.xml</name>
					<originalName>00000001_PREMIS.xml</originalName>
					<pruefalgorithmus>MD5</pruefalgorithmus>
					<pruefsumme>1428a269ff4e5b4894793b68646984b7</pruefsumme>
				</datei>
				<datei id="_MKhAIC639MxzyOn8ji3tN5">
					<name>00000002_PREMIS.xml</name>
					<originalName>00000002_PREMIS.xml</originalName>
					<pruefalgorithmus>MD5</pruefalgorithmus>
					<pruefsumme>f338f61911d2620972b0ac668dcc37ec</pruefsumme>
				</datei>
				<datei id="_fZzi3dX2jvrwakvY6jeJS8">
					<name>Prozess_Digitalisierung_PREMIS.xml</name>
					<originalName>Prozess_Digitalisierung_PREMIS.xml</originalName>
					<pruefalgorithmus>MD5</pruefalgorithmus>
					<pruefsumme>8067daaa900eba6dace69572eea8f8f3</pruefsumme>
				</datei>
				<datei id="_miEf29GTkFR7ymi91IV4fO">
					<name>00000001.jp2</name>
					<originalName>00000001.jp2</originalName>
					<pruefalgorithmus>MD5</pruefalgorithmus>
					<pruefsumme>f7dc1f76a55cbdca0ae4a6dc8ae64644</pruefsumme>
				</datei>
				<datei id="_mOXw3hINt3zY6WvKQOfYmk">
					<name>00000002.jp2</name>
					<originalName>00000002.jp2</originalName>
					<pruefalgorithmus>MD5</pruefalgorithmus>
					<pruefsumme>954d06be4a70c188b6b2e5fe4309fb2c</pruefsumme>
				</datei>
			</ordner>
		</ordner>
	</inhaltsverzeichnis>
</paket>
`

func TestVerifyManifest(t *testing.T) {
	t.Parallel()

	digitizedAIP := fs.NewDir(t, "Test_Digitized_AIP",
		fs.WithDir("additional",
			fs.WithFile("UpdatedAreldaMetadata.xml", manifest),
		),
		fs.WithDir("content",
			fs.WithDir("content",
				fs.WithDir("d_0000001",
					fs.WithFile("00000001.jp2", ""),
					fs.WithFile("00000001_PREMIS.xml", ""),
					fs.WithFile("00000002.jp2", ""),
					fs.WithFile("00000002_PREMIS.xml", ""),
					fs.WithFile("Prozess_Digitalisierung_PREMIS.xml", ""),
				),
			),
			fs.WithDir("header",
				fs.WithDir("old",
					fs.WithDir("SIP",
						fs.WithFile("metadata.xml", ""),
					),
				),
			),
		),
	)
	testSIP, err := sip.New(digitizedAIP.Path())
	assert.NilError(t, err)

	missingFile := fs.NewDir(t, "Test_Missing_File",
		fs.WithDir("additional",
			fs.WithFile("UpdatedAreldaMetadata.xml", manifest),
		),
		fs.WithDir("content",
			fs.WithDir("content",
				fs.WithDir("d_0000001",
					// fs.WithFile("00000001.jp2", ""),
					fs.WithFile("00000001_PREMIS.xml", ""),
					fs.WithFile("00000002.jp2", ""),
					fs.WithFile("00000002_PREMIS.xml", ""),
					fs.WithFile("Prozess_Digitalisierung_PREMIS.xml", ""),
				),
			),
			fs.WithDir("header",
				fs.WithDir("old",
					fs.WithDir("SIP",
						fs.WithFile("metadata.xml", ""),
					),
				),
			),
		),
	)
	missingFileSIP, err := sip.New(missingFile.Path())
	assert.NilError(t, err)

	extraFile := fs.NewDir(t, "Test_Extra_File",
		fs.WithDir("additional",
			fs.WithFile("UpdatedAreldaMetadata.xml", manifest),
		),
		fs.WithDir("content",
			fs.WithDir("content",
				fs.WithDir("d_0000001",
					fs.WithFile("extra_file.txt", "I'm an extra file."),
					fs.WithFile("00000001.jp2", ""),
					fs.WithFile("00000001_PREMIS.xml", ""),
					fs.WithFile("00000002.jp2", ""),
					fs.WithFile("00000002_PREMIS.xml", ""),
					fs.WithFile("Prozess_Digitalisierung_PREMIS.xml", ""),
				),
			),
			fs.WithDir("header",
				fs.WithDir("old",
					fs.WithDir("SIP",
						fs.WithFile("metadata.xml", ""),
					),
				),
			),
		),
	)
	extraFileSIP, err := sip.New(extraFile.Path())
	assert.NilError(t, err)

	tests := []struct {
		name    string
		params  activities.VerifyManifestParams
		result  activities.VerifyManifestResult
		wantErr string
	}{
		{
			name: "Verifies an accurate manifest",
			params: activities.VerifyManifestParams{
				SIP: testSIP,
			},
			result: activities.VerifyManifestResult{},
		},
		{
			name: "Returns a list of missing files",
			params: activities.VerifyManifestParams{
				SIP: missingFileSIP,
			},
			result: activities.VerifyManifestResult{
				Failures: []string{
					fmt.Sprintf(
						"Missing file: %s",
						filepath.Join("d_0000001", "00000001.jp2"),
					),
				},
			},
		},
		{
			name: "Returns a list of extra files",
			params: activities.VerifyManifestParams{
				SIP: extraFileSIP,
			},
			result: activities.VerifyManifestResult{
				Failures: []string{
					fmt.Sprintf(
						"Unexpected file: %s",
						filepath.Join("d_0000001", "extra_file.txt"),
					),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ts := &temporalsdk_testsuite.WorkflowTestSuite{}
			env := ts.NewTestActivityEnvironment()
			env.RegisterActivityWithOptions(
				activities.NewVerifyManifest().Execute,
				temporalsdk_activity.RegisterOptions{Name: activities.VerifyManifestName},
			)

			future, err := env.ExecuteActivity(activities.VerifyManifestName, tt.params)
			if tt.wantErr != "" {
				if err == nil {
					t.Errorf("error is nil, expecting: %q", tt.wantErr)
				} else {
					assert.ErrorContains(t, err, tt.wantErr)
				}

				return
			}
			assert.NilError(t, err)

			var res activities.VerifyManifestResult
			future.Get(&res)
			assert.DeepEqual(t, res, tt.result)
		})
	}
}
