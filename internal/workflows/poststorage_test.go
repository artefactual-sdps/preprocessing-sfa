package workflows_test

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/artefactual-sdps/enduro/pkg/childwf"
	"github.com/artefactual-sdps/temporal-activities/archivezip"
	"github.com/artefactual-sdps/temporal-activities/bucketupload"
	"github.com/artefactual-sdps/temporal-activities/removepaths"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	temporalsdk_activity "go.temporal.io/sdk/activity"
	temporalsdk_testsuite "go.temporal.io/sdk/testsuite"
	temporalsdk_worker "go.temporal.io/sdk/worker"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/ais"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/apis"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/config"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/workflows"
)

type TestSuite struct {
	suite.Suite
	temporalsdk_testsuite.WorkflowTestSuite

	env      *temporalsdk_testsuite.TestWorkflowEnvironment
	workflow *workflows.Poststorage
	testDir  string
}

func (s *TestSuite) setup(cfg *config.PoststorageConfig) {
	s.env = s.NewTestWorkflowEnvironment()
	s.env.SetWorkerOptions(temporalsdk_worker.Options{EnableSessionWorker: true})
	s.testDir = s.T().TempDir()
	cfg.WorkingDir = s.testDir

	s.env.RegisterActivityWithOptions(
		ais.NewGetAIPPathActivity(nil).Execute,
		temporalsdk_activity.RegisterOptions{Name: ais.GetAIPPathActivityName},
	)
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

	s.workflow = workflows.NewPoststorage(*cfg)
}

func TestWorkflow(t *testing.T) {
	suite.Run(t, new(TestSuite))
}

func (s *TestSuite) TestWorkflowSuccess() {
	aipUUID := "9390594f-84c2-457d-bd6a-618f21f7c954"

	s.setup(&config.PoststorageConfig{})
	s.mockActivitiesSuccess(aipUUID)

	s.env.ExecuteWorkflow(
		s.workflow.Execute,
		&childwf.PostStorageParams{
			AIPUUID: aipUUID,
			CustomMetadata: childwf.CustomMetadata{
				apis.CustomMetadataKey: json.RawMessage(
					`{"importTaskId":"task-000001","decision":"Continue and append"}`,
				),
			},
		},
	)

	s.True(s.env.IsWorkflowCompleted())
	s.env.AssertExpectations(s.T())

	var result childwf.PostStorageResult
	err := s.env.GetWorkflowResult(&result)
	s.NoError(err)

	s.Equal(result, childwf.PostStorageResult{})
}

func (s *TestSuite) mockActivitiesSuccess(aipUUID string) {
	aipName := "test-" + aipUUID
	searchMDName := fmt.Sprintf("search-md_%s", aipName)
	localDir := filepath.Join(s.testDir, searchMDName)

	// Mock activities.
	s.env.OnActivity(
		ais.GetAIPPathActivityName,
		mock.AnythingOfType("*context.timerCtx"),
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
