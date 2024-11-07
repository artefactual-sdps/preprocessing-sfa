package ais_test

import (
	"fmt"
	"path/filepath"
	"testing"

	temporalsdk_activity "go.temporal.io/sdk/activity"
	temporalsdk_testsuite "go.temporal.io/sdk/testsuite"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/ais"
)

const (
	arelda = `<?xml version="1.0" encoding="UTF-8"?>
<paket xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
       xmlns:xip="http://www.tessella.com/XIP/v4"
       xmlns="http://bar.admin.ch/arelda/v4"
       xmlns:xs="http://www.w3.org/2001/XMLSchema"
	   xmlns:submissionTests="http://bar.admin.ch/submissionTestResult"
	   xsi:type="paketAIP"
	   schemaVersion="5.0">

	<ablieferung xsi:type="ablieferungFilesAIP">
		<ablieferungstyp>FILES</ablieferungstyp>
		<ablieferndeStelle>Bundesverwaltung (Bern)</ablieferndeStelle>
		<ablieferungsnummer>1000/893_3251903</ablieferungsnummer>
	</ablieferung>
</packet>
`

	mets = `<?xml version='1.0' encoding='UTF-8'?>
<mets:mets xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xlink="http://www.w3.org/1999/xlink" xmlns:mets="http://www.loc.gov/METS/" xsi:schemaLocation="http://www.loc.gov/METS/ http://www.loc.gov/standards/mets/version1121/mets.xsd">
</mets:mets>
`
)

type test struct {
	dir string

	name         string
	params       ais.CombineMDActivityParams
	want         ais.CombineMDActivityResult
	wantErr      string
	wantManifest fs.Manifest
}

type testFunc func(string) test

func testDir(t *testing.T) *fs.Dir {
	return fs.NewDir(t, "ppsfa",
		fs.WithFile("arelda.xml", arelda),
		fs.WithFile("mets.xml", mets),
	)
}

func TestExecute(t *testing.T) {
	t.Parallel()

	for _, tf := range []testFunc{
		func(dir string) test {
			return test{
				dir:  dir,
				name: "Returns the combined metadata",
				params: ais.CombineMDActivityParams{
					AreldaPath: filepath.Join(dir, "arelda.xml"),
					METSPath:   filepath.Join(dir, "mets.xml"),
					LocalDir:   dir,
				},
				want: ais.CombineMDActivityResult{
					Path: filepath.Join(dir, "AIS_1000_893_3251903"),
				},
				wantManifest: fs.Expected(t,
					fs.WithFile("AIS_1000_893_3251903", arelda+mets, fs.WithMode(0o644)),
				),
			}
		},
		func(dir string) test {
			return test{
				dir:  dir,
				name: "Errors if the Arelda file doesn't exist",
				params: ais.CombineMDActivityParams{
					AreldaPath: filepath.Join(dir, "missing.xml"),
					LocalDir:   dir,
				},
				wantErr: fmt.Sprintf(
					"activity error (type: combine-metadata-files, scheduledEventID: 0, startedEventID: 0, identity: ): missing Arelda file: %s/missing.xml",
					dir,
				),
			}
		},
		func(dir string) test {
			return test{
				dir:  dir,
				name: "Errors if the METS file doesn't exist",
				params: ais.CombineMDActivityParams{
					AreldaPath: filepath.Join(dir, "arelda.xml"),
					METSPath:   filepath.Join(dir, "missing.xml"),
					LocalDir:   dir,
				},
				wantErr: fmt.Sprintf(
					"activity error (type: combine-metadata-files, scheduledEventID: 0, startedEventID: 0, identity: ): missing METS file: %s/missing.xml",
					dir,
				),
			}
		},
		func(dir string) test {
			return test{
				dir:  dir,
				name: "Errors when the Arelda file is invalid",
				params: ais.CombineMDActivityParams{
					AreldaPath: filepath.Join(dir, "mets.xml"),
					METSPath:   filepath.Join(dir, "mets.xml"),
					LocalDir:   dir,
				},
				wantErr: "activity error (type: combine-metadata-files, scheduledEventID: 0, startedEventID: 0, identity: ): name AIS file: get accession number: can't find ablieferungsnummer in \"mets.xml\"",
			}
		},
	} {
		tt := tf(testDir(t).Path())
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ts := &temporalsdk_testsuite.WorkflowTestSuite{}
			env := ts.NewTestActivityEnvironment()
			env.RegisterActivityWithOptions(
				ais.NewCombineMDActivity().Execute,
				temporalsdk_activity.RegisterOptions{Name: ais.CombineMDActivityName},
			)

			future, err := env.ExecuteActivity(ais.CombineMDActivityName, tt.params)
			if tt.wantErr != "" {
				if err == nil {
					t.Errorf("error is nil, expecting: %q", tt.wantErr)
				} else {
					assert.ErrorContains(t, err, tt.wantErr)
				}

				return
			}
			assert.NilError(t, err)

			var got ais.CombineMDActivityResult
			future.Get(&got)
			assert.DeepEqual(t, got, tt.want)
			assert.Assert(t, fs.Equal(tt.dir, tt.wantManifest))
		})
	}
}
