package workflow_test

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/artefactual-sdps/temporal-activities/bagcreate"
	"github.com/artefactual-sdps/temporal-activities/bagvalidate"
	"github.com/artefactual-sdps/temporal-activities/ffvalidate"
	"github.com/artefactual-sdps/temporal-activities/xmlvalidate"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	temporalsdk_activity "go.temporal.io/sdk/activity"
	temporalsdk_testsuite "go.temporal.io/sdk/testsuite"
	temporalsdk_worker "go.temporal.io/sdk/worker"
	"gotest.tools/v3/fs"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/activities"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/config"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/enums"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/eventlog"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/localact"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/pips"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/premis"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/sip"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/workflow"
)

const (
	relPath  = "sip"
	manifest = `
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
)

var testTime = time.Date(2024, 6, 6, 15, 8, 39, 0, time.UTC)

type PreprocessingTestSuite struct {
	suite.Suite
	temporalsdk_testsuite.WorkflowTestSuite

	env      *temporalsdk_testsuite.TestWorkflowEnvironment
	workflow *workflow.PreprocessingWorkflow
	testDir  string
	sipPath  string
}

func (s *PreprocessingTestSuite) SetupTest(cfg *config.Configuration) {
	s.env = s.NewTestWorkflowEnvironment()
	s.env.SetStartTime(testTime)
	s.env.SetWorkerOptions(temporalsdk_worker.Options{EnableSessionWorker: true})
	s.testDir = s.T().TempDir()
	cfg.SharedPath = s.testDir

	sp := filepath.Join(s.testDir, relPath)
	if err := os.Mkdir(sp, os.FileMode(0o700)); err != nil {
		s.T().Fatalf("create sip dir: %v", err)
	}
	s.sipPath = sp

	// Register activities.
	s.env.RegisterActivityWithOptions(
		bagvalidate.New(nil).Execute,
		temporalsdk_activity.RegisterOptions{Name: bagvalidate.Name},
	)
	s.env.RegisterActivityWithOptions(
		activities.NewUnbag().Execute,
		temporalsdk_activity.RegisterOptions{Name: activities.UnbagName},
	)
	s.env.RegisterActivityWithOptions(
		activities.NewIdentifySIP().Execute,
		temporalsdk_activity.RegisterOptions{Name: activities.IdentifySIPName},
	)
	s.env.RegisterActivityWithOptions(
		activities.NewValidateStructure().Execute,
		temporalsdk_activity.RegisterOptions{Name: activities.ValidateStructureName},
	)
	s.env.RegisterActivityWithOptions(
		activities.NewVerifyManifest().Execute,
		temporalsdk_activity.RegisterOptions{Name: activities.VerifyManifestName},
	)
	s.env.RegisterActivityWithOptions(
		ffvalidate.New(ffvalidate.Config{}).Execute,
		temporalsdk_activity.RegisterOptions{Name: ffvalidate.Name},
	)
	s.env.RegisterActivityWithOptions(
		activities.NewValidateFiles(nil).Execute,
		temporalsdk_activity.RegisterOptions{Name: activities.ValidateFilesName},
	)
	s.env.RegisterActivityWithOptions(
		activities.NewAddPREMISObjects(rand.Reader).Execute,
		temporalsdk_activity.RegisterOptions{Name: activities.AddPREMISObjectsName},
	)
	s.env.RegisterActivityWithOptions(
		activities.NewAddPREMISEvent().Execute,
		temporalsdk_activity.RegisterOptions{Name: activities.AddPREMISEventName},
	)
	s.env.RegisterActivityWithOptions(
		activities.NewAddPREMISAgent().Execute,
		temporalsdk_activity.RegisterOptions{Name: activities.AddPREMISAgentName},
	)
	s.env.RegisterActivityWithOptions(
		xmlvalidate.New(nil).Execute,
		temporalsdk_activity.RegisterOptions{Name: xmlvalidate.Name},
	)
	s.env.RegisterActivityWithOptions(
		activities.NewValidatePREMIS(nil).Execute,
		temporalsdk_activity.RegisterOptions{Name: activities.ValidatePREMISName},
	)
	s.env.RegisterActivityWithOptions(
		activities.NewTransformSIP().Execute,
		temporalsdk_activity.RegisterOptions{Name: activities.TransformSIPName},
	)
	s.env.RegisterActivityWithOptions(
		activities.NewWriteIdentifierFile().Execute,
		temporalsdk_activity.RegisterOptions{Name: activities.WriteIdentifierFileName},
	)
	s.env.RegisterActivityWithOptions(
		bagcreate.New(cfg.Bagit).Execute,
		temporalsdk_activity.RegisterOptions{Name: bagcreate.Name},
	)

	s.workflow = workflow.NewPreprocessingWorkflow(s.testDir)
}

func (s *PreprocessingTestSuite) digitizedAIP(path string) sip.SIP {
	fs.Apply(
		s.T(),
		fs.DirFromPath(s.T(), path),
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

	r, err := sip.New(path)
	if err != nil {
		s.T().Fatalf("Couldn't create SIP: %v", err)
	}

	return r
}

func (s *PreprocessingTestSuite) AfterTest(suiteName, testName string) {
	s.env.AssertExpectations(s.T())
}

func TestPreprocessingWorkflow(t *testing.T) {
	suite.Run(t, new(PreprocessingTestSuite))
}

func (s *PreprocessingTestSuite) writeBagitTxt(path string) {
	if err := os.WriteFile(
		filepath.Join(path, "bagit.txt"),
		[]byte(`
BagIt-Version: 0.97
Tag-File-Character-Encoding: UTF-8
`),
		os.FileMode(0o640),
	); err != nil {
		s.T().Fatalf("write bagit.txt: %v", err)
	}
}

func (s *PreprocessingTestSuite) TestPreprocessingWorkflowSuccess() {
	s.SetupTest(&config.Configuration{})
	s.writeBagitTxt(s.sipPath)

	expectedSIP := s.digitizedAIP(s.sipPath)
	expectedPIP := pips.NewFromSIP(expectedSIP)

	// Mock activities.
	ctx := mock.AnythingOfType("*context.valueCtx")
	sessionCtx := mock.AnythingOfType("*context.timerCtx")
	premisFilePath := filepath.Join(expectedSIP.Path, "metadata", "premis.xml")

	s.env.OnActivity(
		localact.IsBag,
		ctx,
		&localact.IsBagParams{Path: s.sipPath},
	).Return(
		&localact.IsBagResult{IsBag: true}, nil,
	)
	s.env.OnActivity(
		bagvalidate.Name,
		sessionCtx,
		&bagvalidate.Params{Path: s.sipPath},
	).Return(
		&bagvalidate.Result{Valid: true}, nil,
	)
	s.env.OnActivity(
		activities.UnbagName,
		sessionCtx,
		&activities.UnbagParams{Path: s.sipPath},
	).Return(
		&activities.UnbagResult{Path: s.sipPath}, nil,
	)
	s.env.OnActivity(
		activities.IdentifySIPName,
		sessionCtx,
		&activities.IdentifySIPParams{Path: s.sipPath},
	).Return(
		&activities.IdentifySIPResult{SIP: expectedSIP}, nil,
	)
	s.env.OnActivity(
		activities.ValidateStructureName,
		sessionCtx,
		&activities.ValidateStructureParams{SIP: expectedSIP},
	).Return(
		&activities.ValidateStructureResult{}, nil,
	)
	s.env.OnActivity(
		activities.VerifyManifestName,
		sessionCtx,
		&activities.VerifyManifestParams{SIP: expectedSIP},
	).Return(
		&activities.VerifyManifestResult{}, nil,
	)
	s.env.OnActivity(
		ffvalidate.Name,
		sessionCtx,
		&ffvalidate.Params{Path: expectedSIP.ContentPath},
	).Return(
		&ffvalidate.Result{}, nil,
	)
	s.env.OnActivity(
		activities.ValidateFilesName,
		sessionCtx,
		&activities.ValidateFilesParams{SIP: expectedSIP},
	).Return(
		&activities.ValidateFilesResult{}, nil,
	)
	s.env.OnActivity(
		xmlvalidate.Name,
		sessionCtx,
		&xmlvalidate.Params{
			XMLPath: expectedSIP.ManifestPath,
			XSDPath: expectedSIP.XSDPath,
		},
	).Return(
		&xmlvalidate.Result{}, nil,
	)

	s.env.OnActivity(
		activities.ValidatePREMISName,
		sessionCtx,
		&activities.ValidatePREMISParams{Path: expectedSIP.LogicalMDPath},
	).Return(
		&activities.ValidatePREMISResult{}, nil,
	)

	// PREMIS activities.
	s.env.OnActivity(
		activities.AddPREMISObjectsName,
		sessionCtx,
		&activities.AddPREMISObjectsParams{
			SIP:            expectedSIP,
			PREMISFilePath: premisFilePath,
		},
	).Return(
		&activities.AddPREMISObjectsResult{}, nil,
	)
	s.env.OnActivity(
		activities.AddPREMISEventName,
		sessionCtx,
		&activities.AddPREMISEventParams{
			PREMISFilePath: premisFilePath,
			Agent:          premis.AgentDefault(),
			Type:           "validation",
			Detail:         "name=\"Validate SIP structure\"",
			OutcomeDetail:  "SIP structure identified: DigitizedAIP. SIP structure matches validation criteria.",
			Failures:       nil,
		},
	).Return(
		&activities.AddPREMISEventResult{}, nil,
	)
	s.env.OnActivity(
		activities.AddPREMISEventName,
		sessionCtx,
		&activities.AddPREMISEventParams{
			PREMISFilePath: premisFilePath,
			Agent:          premis.AgentDefault(),
			Type:           "validation",
			Detail:         "name=\"Validate file format\"",
			OutcomeDetail:  "Format allowed",
			Failures:       nil,
		},
	).Return(
		&activities.AddPREMISEventResult{}, nil,
	)
	s.env.OnActivity(
		activities.AddPREMISEventName,
		sessionCtx,
		&activities.AddPREMISEventParams{
			PREMISFilePath: premisFilePath,
			Agent:          premis.AgentDefault(),
			Type:           "validation",
			Detail:         "name=\"Validate SIP metadata\"",
			OutcomeDetail:  "Metadata validation successful",
			Failures:       nil,
		},
	).Return(
		&activities.AddPREMISEventResult{}, nil,
	)
	s.env.OnActivity(
		activities.AddPREMISAgentName,
		sessionCtx,
		&activities.AddPREMISAgentParams{
			PREMISFilePath: premisFilePath,
			Agent:          premis.AgentDefault(),
		},
	).Return(
		&activities.AddPREMISAgentResult{}, nil,
	)

	// Transform SIP.
	s.env.OnActivity(
		activities.TransformSIPName,
		sessionCtx,
		&activities.TransformSIPParams{SIP: expectedSIP},
	).Return(
		&activities.TransformSIPResult{PIP: expectedPIP}, nil,
	)

	s.env.OnActivity(
		activities.WriteIdentifierFileName,
		sessionCtx,
		&activities.WriteIdentifierFileParams{PIP: expectedPIP},
	).Return(
		&activities.WriteIdentifierFileResult{
			Path: filepath.Join(s.sipPath, "metadata", "identifiers.json"),
		}, nil,
	)

	s.env.OnActivity(
		bagcreate.Name,
		sessionCtx,
		&bagcreate.Params{SourcePath: s.sipPath},
	).Return(
		&bagcreate.Result{}, nil,
	)

	s.env.ExecuteWorkflow(
		s.workflow.Execute,
		&workflow.PreprocessingWorkflowParams{RelativePath: relPath},
	)

	s.True(s.env.IsWorkflowCompleted())

	var result workflow.PreprocessingWorkflowResult
	err := s.env.GetWorkflowResult(&result)
	s.NoError(err)
	s.Equal(
		&workflow.PreprocessingWorkflowResult{
			Outcome:      workflow.OutcomeSuccess,
			RelativePath: relPath,
			PreservationTasks: []*eventlog.Event{
				{
					Name:        "Validate Bag",
					Message:     "Bag validated",
					Outcome:     enums.EventOutcomeSuccess,
					StartedAt:   testTime,
					CompletedAt: testTime,
				},
				{
					Name:        "Unbag SIP",
					Message:     "SIP unbagged",
					Outcome:     enums.EventOutcomeSuccess,
					StartedAt:   testTime,
					CompletedAt: testTime,
				},
				{
					Name:        "Identify SIP structure",
					Message:     "SIP structure identified: DigitizedAIP",
					Outcome:     enums.EventOutcomeSuccess,
					StartedAt:   testTime,
					CompletedAt: testTime,
				},
				{
					Name:        "Validate SIP structure",
					Message:     "SIP structure matches validation criteria",
					Outcome:     enums.EventOutcomeSuccess,
					StartedAt:   testTime,
					CompletedAt: testTime,
				},
				{
					Name:        "Verify SIP manifest",
					Message:     "SIP contents match manifest",
					Outcome:     enums.EventOutcomeSuccess,
					StartedAt:   testTime,
					CompletedAt: testTime,
				},
				{
					Name:        "Verify SIP checksums",
					Message:     "SIP checksums match file contents",
					Outcome:     enums.EventOutcomeSuccess,
					StartedAt:   testTime,
					CompletedAt: testTime,
				},
				{
					Name:        "Validate SIP file formats",
					Message:     "No disallowed file formats found",
					Outcome:     enums.EventOutcomeSuccess,
					StartedAt:   testTime,
					CompletedAt: testTime,
				},
				{
					Name:        "Validate SIP files",
					Message:     "No invalid files found",
					Outcome:     enums.EventOutcomeSuccess,
					StartedAt:   testTime,
					CompletedAt: testTime,
				},
				{
					Name:        "Validate SIP metadata",
					Message:     "Metadata validation successful",
					Outcome:     enums.EventOutcomeSuccess,
					StartedAt:   testTime,
					CompletedAt: testTime,
				},
				{
					Name:        "Validate logical metadata",
					Message:     "Logical metadata validation successful",
					Outcome:     enums.EventOutcomeSuccess,
					StartedAt:   testTime,
					CompletedAt: testTime,
				},
				{
					Name:        "Create premis.xml",
					Message:     "Created a premis.xml and stored in metadata directory",
					Outcome:     enums.EventOutcomeSuccess,
					StartedAt:   testTime,
					CompletedAt: testTime,
				},
				{
					Name:        "Restructure SIP",
					Message:     "SIP has been restructured",
					Outcome:     enums.EventOutcomeSuccess,
					StartedAt:   testTime,
					CompletedAt: testTime,
				},
				{
					Name:        "Create identifier.json",
					Message:     "Created an identifier.json and stored in metadata directory",
					Outcome:     enums.EventOutcomeSuccess,
					StartedAt:   testTime,
					CompletedAt: testTime,
				},
				{
					Name:        "Bag SIP",
					Message:     "SIP has been bagged",
					Outcome:     enums.EventOutcomeSuccess,
					StartedAt:   testTime,
					CompletedAt: testTime,
				},
			},
		},
		&result,
	)
}

func (s *PreprocessingTestSuite) TestPreprocessingWorkflowIdentifySIPFails() {
	s.SetupTest(&config.Configuration{})
	sipPath := filepath.Join(s.testDir, relPath)

	// Mock activities.
	sessionCtx := mock.AnythingOfType("*context.timerCtx")
	s.env.OnActivity(
		activities.IdentifySIPName,
		sessionCtx,
		&activities.IdentifySIPParams{Path: sipPath},
	).Return(
		nil, fmt.Errorf("IdentifySIP: NewSIP: stat : no such file or directory"),
	)

	s.env.ExecuteWorkflow(
		s.workflow.Execute,
		&workflow.PreprocessingWorkflowParams{RelativePath: relPath},
	)

	s.True(s.env.IsWorkflowCompleted())

	var result workflow.PreprocessingWorkflowResult
	err := s.env.GetWorkflowResult(&result)
	s.NoError(err)
	s.Equal(
		&workflow.PreprocessingWorkflowResult{
			Outcome:      workflow.OutcomeSystemError,
			RelativePath: relPath,
			PreservationTasks: []*eventlog.Event{
				{
					Name:        "Identify SIP structure",
					Message:     "System error: SIP structure identification has failed",
					Outcome:     enums.EventOutcomeSystemFailure,
					StartedAt:   testTime,
					CompletedAt: testTime,
				},
			},
		},
		&result,
	)
}

func (s *PreprocessingTestSuite) TestPreprocessingWorkflowValidationFails() {
	s.SetupTest(&config.Configuration{})

	sipPath := filepath.Join(s.testDir, relPath)
	expectedSIP := s.digitizedAIP(sipPath)

	// Mock activities.
	sessionCtx := mock.AnythingOfType("*context.timerCtx")
	s.env.OnActivity(
		activities.IdentifySIPName,
		sessionCtx,
		&activities.IdentifySIPParams{Path: sipPath},
	).Return(
		&activities.IdentifySIPResult{SIP: expectedSIP}, nil,
	)
	s.env.OnActivity(
		activities.ValidateStructureName,
		sessionCtx,
		&activities.ValidateStructureParams{SIP: expectedSIP},
	).Return(
		&activities.ValidateStructureResult{
			Failures: []string{
				"XSD folder is missing",
				"UpdatedAreldaMetadata.xml is missing",
			},
		},
		nil,
	)
	s.env.OnActivity(
		activities.VerifyManifestName,
		sessionCtx,
		&activities.VerifyManifestParams{SIP: expectedSIP},
	).Return(
		&activities.VerifyManifestResult{
			Failed: true,
			ChecksumFailures: []string{
				`Checksum mismatch for "content/content/d_0000001/00000001.jp2" (expected: "827ccb0eea8a706c4c34a16891f84e7b", got: "2714364e3a0ac68e8bf9b898b31ff303")`,
			},
			MissingFiles:    []string{"Missing file: d_0000001/00000001.jp2"},
			UnexpectedFiles: []string{"Unexpected file: d_0000001/extra_file.txt"},
		},
		nil,
	)
	s.env.OnActivity(
		ffvalidate.Name,
		sessionCtx,
		&ffvalidate.Params{Path: expectedSIP.ContentPath},
	).Return(
		&ffvalidate.Result{Failures: []string{
			`file format fmt/11 not allowed: "content/content/d_0000001/00000010.png"`,
			`file format fmt/11 not allowed: "content/content/d_0000001/00000011.png"`,
		}},
		nil,
	)
	s.env.OnActivity(
		activities.ValidateFilesName,
		sessionCtx,
		&activities.ValidateFilesParams{SIP: expectedSIP},
	).Return(
		&activities.ValidateFilesResult{
			Failures: []string{`invalid PDF/A: "contents/contents/d_0000001/test.pdf"`},
		},
		nil,
	)
	s.env.OnActivity(
		xmlvalidate.Name,
		sessionCtx,
		&xmlvalidate.Params{
			XMLPath: expectedSIP.ManifestPath,
			XSDPath: expectedSIP.XSDPath,
		},
	).Return(
		&xmlvalidate.Result{
			Failures: []string{
				`additional/UpdatedAreldaMetadata.xml does not match expected metadata requirements`,
			},
		}, nil,
	)

	s.env.OnActivity(
		activities.ValidatePREMISName,
		sessionCtx,
		&activities.ValidatePREMISParams{Path: expectedSIP.LogicalMDPath},
	).Return(
		&activities.ValidatePREMISResult{
			Failures: []string{`test-AIP-premis.xml does not match expected metadata requirements`},
		}, nil,
	)

	s.env.ExecuteWorkflow(
		s.workflow.Execute,
		&workflow.PreprocessingWorkflowParams{RelativePath: relPath},
	)

	s.True(s.env.IsWorkflowCompleted())

	var result workflow.PreprocessingWorkflowResult
	err := s.env.GetWorkflowResult(&result)
	s.NoError(err)
	s.Equal(
		&workflow.PreprocessingWorkflowResult{
			Outcome:      workflow.OutcomeContentError,
			RelativePath: relPath,
			PreservationTasks: []*eventlog.Event{
				{
					Name:        "Identify SIP structure",
					Message:     "SIP structure identified: DigitizedAIP",
					Outcome:     enums.EventOutcomeSuccess,
					StartedAt:   testTime,
					CompletedAt: testTime,
				},
				{
					Name: "Validate SIP structure",
					Message: `Content error: SIP structure validation has failed:
XSD folder is missing
UpdatedAreldaMetadata.xml is missing`,
					Outcome:     enums.EventOutcomeValidationFailure,
					StartedAt:   testTime,
					CompletedAt: testTime,
				},
				{
					Name: "Verify SIP manifest",
					Message: `Content error: SIP contents do not match "UpdatedAreldaMetadata.xml":
Missing file: d_0000001/00000001.jp2
Unexpected file: d_0000001/extra_file.txt`,
					Outcome:     enums.EventOutcomeValidationFailure,
					StartedAt:   testTime,
					CompletedAt: testTime,
				},
				{
					Name: "Verify SIP checksums",
					Message: `Content error: SIP checksums do not match file contents:
Checksum mismatch for "content/content/d_0000001/00000001.jp2" (expected: "827ccb0eea8a706c4c34a16891f84e7b", got: "2714364e3a0ac68e8bf9b898b31ff303")`,
					Outcome:     enums.EventOutcomeValidationFailure,
					StartedAt:   testTime,
					CompletedAt: testTime,
				},
				{
					Name: "Validate SIP file formats",
					Message: `Content error: file format validation has failed. One or more file formats are not allowed:
file format fmt/11 not allowed: "content/content/d_0000001/00000010.png"
file format fmt/11 not allowed: "content/content/d_0000001/00000011.png"`,
					Outcome:     enums.EventOutcomeValidationFailure,
					StartedAt:   testTime,
					CompletedAt: testTime,
				},
				{
					Name: "Validate SIP files",
					Message: `Content error: file validation has failed. One or more files are invalid:
invalid PDF/A: "contents/contents/d_0000001/test.pdf"`,
					Outcome:     enums.EventOutcomeValidationFailure,
					StartedAt:   testTime,
					CompletedAt: testTime,
				},
				{
					Name:        "Validate SIP metadata",
					Message:     "Content error: metadata validation has failed:\nadditional/UpdatedAreldaMetadata.xml does not match expected metadata requirements",
					Outcome:     enums.EventOutcomeValidationFailure,
					StartedAt:   testTime,
					CompletedAt: testTime,
				},
				{
					Name: "Validate logical metadata",
					Message: `Content error: logical metadata validation has failed:
test-AIP-premis.xml does not match expected metadata requirements`,
					Outcome:     enums.EventOutcomeValidationFailure,
					StartedAt:   testTime,
					CompletedAt: testTime,
				},
			},
		},
		&result,
	)
}
