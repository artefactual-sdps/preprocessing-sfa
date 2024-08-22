package activities_test

import (
	"testing"

	temporalsdk_activity "go.temporal.io/sdk/activity"
	temporalsdk_testsuite "go.temporal.io/sdk/testsuite"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/activities"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/sip"
)

const (
	aipManifest = `
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

	sipManifest = `
<?xml version="1.0" encoding="UTF-8"?>
<paket
	xmlns="http://bar.admin.ch/arelda/v4"
	xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" schemaVersion="4.0" xsi:type="paketSIP">
	<paketTyp>SIP</paketTyp>
	<inhaltsverzeichnis>
		<ordner>
			<name>header</name>
			<originalName>header</originalName>
			<ordner>
				<name>xsd</name>
				<originalName>xsd</originalName>
				<datei id="_ZSANrSklQ9HGn99yjlUumz">
					<name>arelda.xsd</name>
					<originalName>arelda.xsd</originalName>
					<pruefalgorithmus>MD5</pruefalgorithmus>
					<pruefsumme>f8454632e1ebf97e0aa8d9527ce2641f</pruefsumme>
				</datei>
			</ordner>
		</ordner>
		<ordner>
			<name>content</name>
			<originalName>content</originalName>
			<ordner>
				<name>d_0000001</name>
				<originalName>d_0000001</originalName>
				<datei id="_zodSTSD0nv05CpOp6JoV3X">
					<name>00000001.jp2</name>
					<originalName>00000001.jp2</originalName>
					<pruefalgorithmus>MD5</pruefalgorithmus>
					<pruefsumme>dc29291d0e2a18363d0efd2ec2fe81c9</pruefsumme>
				</datei>
				<datei id="_rlPKJX9ZcAl4ooc4IfoIkM">
					<name>00000002.jp2</name>
					<originalName>00000002.jp2</originalName>
					<pruefalgorithmus>MD5</pruefalgorithmus>
					<pruefsumme>9093907ec32f06fe595e0f14982c4bf0</pruefsumme>
				</datei>
				<datei id="_WuDmXAs5UDwKTGVLsCcZxa">
					<name>00000001_PREMIS.xml</name>
					<originalName>00000001_PREMIS.xml</originalName>
					<pruefalgorithmus>MD5</pruefalgorithmus>
					<pruefsumme>1d310772d26138a42eb2d6bebb637457</pruefsumme>
				</datei>
				<datei id="_Ohk77y2DJa82RXqsWG4S90">
					<name>00000002_PREMIS.xml</name>
					<originalName>00000002_PREMIS.xml</originalName>
					<pruefalgorithmus>MD5</pruefalgorithmus>
					<pruefsumme>abe7d286e9fa8db7ab8a3078df761c8e</pruefsumme>
				</datei>
				<datei id="_cQ6sm5CChWVqtqmrWvne0W">
					<name>Prozess_Digitalisierung_PREMIS.xml</name>
					<originalName>Prozess_Digitalisierung_PREMIS.xml</originalName>
					<pruefalgorithmus>MD5</pruefalgorithmus>
					<pruefsumme>21d8e90afdefd2c43386ca1d1658cab0</pruefsumme>
				</datei>
			</ordner>
		</ordner>
	</inhaltsverzeichnis>
</paket>
`
)

func testSIP(t *testing.T, dir *fs.Dir) sip.SIP {
	t.Helper()
	s, err := sip.New(dir.Path())
	if err != nil {
		t.Fatalf("sip: New(): %v", err)
	}
	return s
}

func TestVerifyManifest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		params  activities.VerifyManifestParams
		want    activities.VerifyManifestResult
		wantErr string
	}{
		{
			name: "Verifies a digitized AIP manifest",
			params: activities.VerifyManifestParams{
				SIP: testSIP(
					t,
					fs.NewDir(t, "Test_Digitized_AIP",
						fs.WithDir("additional",
							fs.WithFile("UpdatedAreldaMetadata.xml", aipManifest),
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
								fs.WithDir("xsd",
									fs.WithFile("arelda.xsd", ""),
								),
							),
						),
					),
				),
			},
			want: activities.VerifyManifestResult{},
		},
		{
			name: "Verifies a digitized SIP manifest",
			params: activities.VerifyManifestParams{
				SIP: testSIP(
					t,
					fs.NewDir(t, "Test_Digitized_SIP",
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
							fs.WithDir("xsd",
								fs.WithFile("arelda.xsd", ""),
							),
							fs.WithFile("metadata.xml", sipManifest),
						),
					),
				),
			},
			want: activities.VerifyManifestResult{},
		},
		{
			name: "Returns a list of missing files",
			params: activities.VerifyManifestParams{
				SIP: testSIP(
					t,
					fs.NewDir(t, "Test_Missing_Files",
						fs.WithDir("additional",
							fs.WithFile("UpdatedAreldaMetadata.xml", aipManifest),
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
					),
				),
			},
			want: activities.VerifyManifestResult{
				Failures: []string{
					"Missing file: content/content/d_0000001/00000001.jp2",
					"Missing file: content/header/xsd/arelda.xsd",
				},
			},
		},
		{
			name: "Returns a list of extra files",
			params: activities.VerifyManifestParams{
				SIP: testSIP(
					t,
					fs.NewDir(t, "Test_Extra_Files",
						fs.WithDir("additional",
							fs.WithFile("UpdatedAreldaMetadata.xml", aipManifest),
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
								fs.WithDir("xsd",
									fs.WithFile("arelda.xsd", ""),
									fs.WithFile("extra.xsd", ""),
								),
							),
						),
					),
				),
			},
			want: activities.VerifyManifestResult{
				Failures: []string{
					"Unexpected file: content/content/d_0000001/extra_file.txt",
					"Unexpected file: content/header/xsd/extra.xsd",
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
			assert.DeepEqual(t, res, tt.want)
		})
	}
}
