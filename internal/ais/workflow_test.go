package ais_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/artefactual-sdps/temporal-activities/archivezip"
	"github.com/artefactual-sdps/temporal-activities/bucketupload"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	temporalsdk_testsuite "go.temporal.io/sdk/testsuite"
	temporalsdk_worker "go.temporal.io/sdk/worker"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/ais"
)

type TestSuite struct {
	suite.Suite
	temporalsdk_testsuite.WorkflowTestSuite

	env      *temporalsdk_testsuite.TestWorkflowEnvironment
	workflow *ais.Workflow
	testDir  string
}

func (s *TestSuite) setup(cfg *ais.Config) {
	s.env = s.NewTestWorkflowEnvironment()
	s.env.SetWorkerOptions(temporalsdk_worker.Options{EnableSessionWorker: true})
	s.testDir = s.T().TempDir()
	cfg.WorkingDir = s.testDir

	ais.RegisterActivities(s.env, nil, nil)

	s.workflow = ais.NewWorkflow(*cfg, nil)
}

func TestWorkflow(t *testing.T) {
	suite.Run(t, new(TestSuite))
}

func (s *TestSuite) TestWorkflowSuccess() {
	aipUUID := "9390594f-84c2-457d-bd6a-618f21f7c954"

	s.setup(&ais.Config{})
	s.mockActivitiesSuccess(aipUUID)

	s.env.ExecuteWorkflow(
		s.workflow.Execute,
		&ais.WorkflowParams{AIPUUID: aipUUID},
	)

	s.True(s.env.IsWorkflowCompleted())
	s.env.AssertExpectations(s.T())

	var result ais.WorkflowResult
	err := s.env.GetWorkflowResult(&result)
	s.NoError(err)

	s.Equal(result, ais.WorkflowResult{Key: "search-md_test-" + aipUUID + ".zip"})
}

func (s *TestSuite) mockActivitiesSuccess(aipUUID string) {
	aipName := "test-" + aipUUID
	searchMDName := fmt.Sprintf("search-md_%s", aipName)
	localDir := filepath.Join(s.testDir, searchMDName)

	// Mock activities.
	s.env.OnActivity(
		ais.GetAIPPathActivity,
		mock.AnythingOfType("*context.valueCtx"),
		&ais.GetAIPPathActivityParams{AIPUUID: aipUUID},
	).Return(
		&ais.GetAIPPathActivityResult{
			Path: "9390/594f/84c2/457d/bd6a/618f/21f7/c954/test-9390594f-84c2-457d-bd6a-618f21f7c954.zip",
		}, nil,
	)

	// Mock session activities.
	sessionCtx := mock.AnythingOfType("*context.timerCtx")
	metsName := fmt.Sprintf("METS.%s.xml", aipUUID)
	metsPath := filepath.Join(localDir, metsName)
	s.env.OnActivity(
		ais.FetchActivityName,
		sessionCtx,
		&ais.FetchActivityParams{
			AIPUUID:      aipUUID,
			RelativePath: fmt.Sprintf("%s/data/%s", aipName, metsName),
			Destination:  metsPath,
		},
	).Return(
		&ais.FetchActivityResult{}, nil,
	)

	mdRelPath := "objects/header/metadata.xml"
	s.env.OnActivity(
		ais.ParseActivityName,
		sessionCtx,
		&ais.ParseActivityParams{METSPath: metsPath},
	).Return(
		&ais.ParseActivityResult{MetadataRelPath: mdRelPath}, nil,
	)

	areldaPath := filepath.Join(localDir, "metadata.xml")
	s.env.OnActivity(
		ais.FetchActivityName,
		sessionCtx,
		&ais.FetchActivityParams{
			AIPUUID:      aipUUID,
			RelativePath: fmt.Sprintf("%s/data/%s", aipName, mdRelPath),
			Destination:  areldaPath,
		},
	).Return(
		&ais.FetchActivityResult{}, nil,
	)

	s.env.OnActivity(
		ais.CombineMDActivityName,
		sessionCtx,
		ais.CombineMDActivityParams{
			AreldaPath: areldaPath,
			METSPath:   metsPath,
			LocalDir:   localDir,
		},
	).Return(
		&ais.CombineMDActivityResult{Path: filepath.Join(localDir, "AIS_1000_893_3251903")}, nil,
	)

	zipPath := localDir + ".zip"
	s.env.OnActivity(
		archivezip.Name,
		sessionCtx,
		&archivezip.Params{SourceDir: localDir},
	).Return(
		&archivezip.Result{Path: zipPath}, nil,
	)

	s.env.OnActivity(
		bucketupload.Name,
		sessionCtx,
		&bucketupload.Params{Path: zipPath},
	).Return(
		&bucketupload.Result{Key: searchMDName + ".zip"}, nil,
	)
}
