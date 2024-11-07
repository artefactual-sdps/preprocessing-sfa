package ais_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/artefactual-sdps/temporal-activities/archivezip"
	"github.com/artefactual-sdps/temporal-activities/bucketupload"
	"github.com/artefactual-sdps/temporal-activities/removepaths"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	temporalsdk_activity "go.temporal.io/sdk/activity"
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

	s.registerActivities()

	s.workflow = ais.NewWorkflow(*cfg, nil)
}

func (s *TestSuite) registerActivities() {
	s.env.RegisterActivityWithOptions(
		ais.NewFetchActivity(nil).Execute,
		temporalsdk_activity.RegisterOptions{Name: ais.FetchActivityName},
	)
	s.env.RegisterActivityWithOptions(
		ais.NewParseActivity().Execute,
		temporalsdk_activity.RegisterOptions{Name: ais.ParseActivityName},
	)
	s.env.RegisterActivityWithOptions(
		ais.NewCombineMDActivity().Execute,
		temporalsdk_activity.RegisterOptions{Name: ais.CombineMDActivityName},
	)
	s.env.RegisterActivityWithOptions(
		archivezip.New().Execute,
		temporalsdk_activity.RegisterOptions{Name: archivezip.Name},
	)
	s.env.RegisterActivityWithOptions(
		bucketupload.New(nil).Execute,
		temporalsdk_activity.RegisterOptions{Name: bucketupload.Name},
	)
	s.env.RegisterActivityWithOptions(
		removepaths.New().Execute,
		temporalsdk_activity.RegisterOptions{Name: removepaths.Name},
	)
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

	s.Equal(result, ais.WorkflowResult{Key: "test-" + aipUUID + ".zip"})
}

func (s *TestSuite) mockActivitiesSuccess(aipUUID string) {
	aipName := "test-" + aipUUID
	aipPath := "9390/594f/84c2/457d/bd6a/618f/21f7/c954/test-9390594f-84c2-457d-bd6a-618f21f7c954.zip"
	localDir := filepath.Join(s.testDir, fmt.Sprintf("search-md_%s", aipName))
	metsName := fmt.Sprintf("METS.%s.xml", aipUUID)
	metsPath := filepath.Join(localDir, metsName)

	// Mock activities.
	s.env.OnActivity(
		ais.GetAIPPathActivity,
		mock.AnythingOfType("*context.valueCtx"),
		&ais.GetAIPPathActivityParams{AIPUUID: aipUUID},
	).Return(
		&ais.GetAIPPathActivityResult{Path: aipPath}, nil,
	)

	// Mock session activities.
	sessionCtx := mock.AnythingOfType("*context.timerCtx")
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

	mdpath := "objects/header/metadata.xml"
	s.env.OnActivity(
		ais.ParseActivityName,
		sessionCtx,
		&ais.ParseActivityParams{METSPath: metsPath},
	).Return(
		&ais.ParseActivityResult{MetadataRelPath: mdpath}, nil,
	)

	areldaPath := filepath.Join(localDir, "metadata.xml")
	s.env.OnActivity(
		ais.FetchActivityName,
		sessionCtx,
		&ais.FetchActivityParams{
			AIPUUID:      aipUUID,
			RelativePath: fmt.Sprintf("%s/data/%s", aipName, mdpath),
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

	zipPath := filepath.Join(s.testDir, aipName+".zip")
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
		&bucketupload.Result{Key: aipName + ".zip"}, nil,
	)
}
