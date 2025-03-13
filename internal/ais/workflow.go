package ais

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/artefactual-sdps/temporal-activities/archivezip"
	"github.com/artefactual-sdps/temporal-activities/bucketupload"
	"github.com/artefactual-sdps/temporal-activities/removepaths"
	temporalsdk_activity "go.temporal.io/sdk/activity"
	temporalsdk_temporal "go.temporal.io/sdk/temporal"
	temporalsdk_worker "go.temporal.io/sdk/worker"
	temporalsdk_workflow "go.temporal.io/sdk/workflow"
	"gocloud.dev/blob"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/amss"
)

type WorkflowParams struct {
	AIPUUID string
}

type WorkflowResult struct {
	Key string
}

type Workflow struct {
	config Config
}

func NewWorkflow(config Config) *Workflow {
	return &Workflow{config: config}
}

func (w *Workflow) Execute(ctx temporalsdk_workflow.Context, params *WorkflowParams) (r *WorkflowResult, e error) {
	r = &WorkflowResult{}

	logger := temporalsdk_workflow.GetLogger(ctx)
	logger.Debug("AIS workflow running!", "params", params)

	defer func() {
		logger.Debug("AIS workflow finished!", "result", r, "error", e)
	}()

	var getAIPPathResult GetAIPPathActivityResult
	err := temporalsdk_workflow.ExecuteActivity(
		temporalsdk_workflow.WithActivityOptions(
			ctx,
			temporalsdk_workflow.ActivityOptions{
				ScheduleToCloseTimeout: 10 * time.Minute,
				RetryPolicy: &temporalsdk_temporal.RetryPolicy{
					InitialInterval:    15 * time.Second,
					BackoffCoefficient: 2,
					MaximumInterval:    time.Minute,
					MaximumAttempts:    5,
				},
			},
		),
		GetAIPPathActivityName,
		&GetAIPPathActivityParams{
			AIPUUID: params.AIPUUID,
		},
	).Get(ctx, &getAIPPathResult)
	if err != nil {
		return nil, err
	}

	// Activities running within a session.
	{
		var sessErr error
		maxAttempts := 5

		activityOpts := temporalsdk_workflow.WithActivityOptions(ctx, temporalsdk_workflow.ActivityOptions{
			StartToCloseTimeout: time.Minute,
			TaskQueue:           w.config.Temporal.TaskQueue,
		})
		for attempt := 1; attempt <= maxAttempts; attempt++ {
			sessCtx, err := temporalsdk_workflow.CreateSession(activityOpts, &temporalsdk_workflow.SessionOptions{
				CreationTimeout:  forever,
				ExecutionTimeout: forever,
			})
			if err != nil {
				return nil, fmt.Errorf("error creating session: %v", err)
			}

			r.Key, sessErr = w.SessionHandler(sessCtx, params.AIPUUID, getAIPPathResult.Path)

			// We want to retry the session if it has been canceled as a result
			// of losing the worker but not otherwise. This scenario seems to be
			// identifiable when we have an error but the root context has not
			// been canceled.
			if sessErr != nil &&
				(errors.Is(sessErr, temporalsdk_workflow.ErrSessionFailed) || temporalsdk_temporal.IsCanceledError(sessErr)) {
				// Root context canceled, hence workflow canceled.
				if ctx.Err() == temporalsdk_workflow.ErrCanceled {
					return nil, nil
				}

				logger.Error(
					"Session failed, will retry shortly (10s)...",
					"err", ctx.Err(),
					"attemptFailed", attempt,
					"attemptsLeft", maxAttempts-attempt,
				)

				_ = temporalsdk_workflow.Sleep(ctx, time.Second*10)

				continue
			}

			break
		}

		if sessErr != nil {
			return nil, sessErr
		}
	}

	return r, nil
}

func (w *Workflow) SessionHandler(ctx temporalsdk_workflow.Context, aipUUID, aipPath string) (s string, e error) {
	removePaths := []string{}

	defer func() {
		var removeResult removepaths.Result
		err := temporalsdk_workflow.ExecuteActivity(
			withFilesystemActivityOpts(ctx),
			removepaths.Name,
			&removepaths.Params{Paths: removePaths},
		).Get(ctx, &removeResult)
		if err != nil {
			e = errors.Join(e, err)
		}

		temporalsdk_workflow.CompleteSession(ctx)
	}()

	// In case the AIP is compressed, remove its UUID and the possible
	// extension from the directory/file name, and append the UUID back.
	aipDirName := strings.Split(filepath.Base(aipPath), aipUUID)[0] + aipUUID
	localDir := filepath.Join(w.config.WorkingDir, fmt.Sprintf("search-md_%s", aipDirName))
	metsName := fmt.Sprintf("METS.%s.xml", aipUUID)
	metsPath := filepath.Join(localDir, metsName)

	removePaths = append(removePaths, localDir)

	var fetchMETSResult FetchActivityResult
	e = temporalsdk_workflow.ExecuteActivity(
		withActivityOptsForLongLivedRequest(ctx),
		FetchActivityName,
		&FetchActivityParams{
			AIPUUID:      aipUUID,
			RelativePath: fmt.Sprintf("%s/data/%s", aipDirName, metsName),
			Destination:  metsPath,
		},
	).Get(ctx, &fetchMETSResult)
	if e != nil {
		return "", e
	}

	var parseResult ParseActivityResult
	e = temporalsdk_workflow.ExecuteActivity(
		withFilesystemActivityOpts(ctx),
		ParseActivityName,
		&ParseActivityParams{METSPath: metsPath},
	).Get(ctx, &parseResult)
	if e != nil {
		return "", e
	}

	var metadataRelPath string
	if parseResult.UpdatedAreldaMetadataRelPath != "" {
		metadataRelPath = parseResult.UpdatedAreldaMetadataRelPath
	} else if parseResult.MetadataRelPath != "" {
		metadataRelPath = parseResult.MetadataRelPath
	} else {
		return "", errors.New("UpdatedAreldaMetadata.xml and metadata.xml files not found in METS")
	}

	metadataPath := filepath.Join(localDir, filepath.Base(metadataRelPath))

	var fetchMetadataResult FetchActivityResult
	e = temporalsdk_workflow.ExecuteActivity(
		withActivityOptsForLongLivedRequest(ctx),
		FetchActivityName,
		&FetchActivityParams{
			AIPUUID:      aipUUID,
			RelativePath: fmt.Sprintf("%s/data/%s", aipDirName, metadataRelPath),
			Destination:  metadataPath,
		},
	).Get(ctx, &fetchMetadataResult)
	if e != nil {
		return "", e
	}

	var combineMDResult CombineMDActivityResult
	e = temporalsdk_workflow.ExecuteActivity(
		withFilesystemActivityOpts(ctx),
		CombineMDActivityName,
		&CombineMDActivityParams{
			AreldaPath: metadataPath,
			METSPath:   metsPath,
			LocalDir:   localDir,
		},
	).Get(ctx, &combineMDResult)
	if e != nil {
		return "", e
	}

	var zipResult archivezip.Result
	e = temporalsdk_workflow.ExecuteActivity(
		withFilesystemActivityOpts(ctx),
		archivezip.Name,
		&archivezip.Params{SourceDir: localDir},
	).Get(ctx, &zipResult)
	if e != nil {
		return "", e
	}

	removePaths = append(removePaths, zipResult.Path)

	var uploadResult bucketupload.Result
	e = temporalsdk_workflow.ExecuteActivity(
		withActivityOptsForLongLivedRequest(ctx),
		bucketupload.Name,
		&bucketupload.Params{Path: zipResult.Path},
	).Get(ctx, &uploadResult)
	if e != nil {
		return "", e
	}

	return uploadResult.Key, nil
}

func RegisterWorkflow(w temporalsdk_worker.Worker, config Config) {
	w.RegisterWorkflowWithOptions(
		NewWorkflow(config).Execute,
		temporalsdk_workflow.RegisterOptions{Name: config.Temporal.WorkflowName},
	)
}

func RegisterActivities(r temporalsdk_worker.ActivityRegistry, amssClient amss.Client, bucket *blob.Bucket) {
	r.RegisterActivityWithOptions(
		NewGetAIPPathActivity(amssClient).Execute,
		temporalsdk_activity.RegisterOptions{Name: GetAIPPathActivityName},
	)
	r.RegisterActivityWithOptions(
		NewFetchActivity(amssClient).Execute,
		temporalsdk_activity.RegisterOptions{Name: FetchActivityName},
	)
	r.RegisterActivityWithOptions(
		NewParseActivity().Execute,
		temporalsdk_activity.RegisterOptions{Name: ParseActivityName},
	)
	r.RegisterActivityWithOptions(
		NewCombineMDActivity().Execute,
		temporalsdk_activity.RegisterOptions{Name: CombineMDActivityName},
	)
	r.RegisterActivityWithOptions(
		archivezip.New().Execute,
		temporalsdk_activity.RegisterOptions{Name: archivezip.Name},
	)
	r.RegisterActivityWithOptions(
		bucketupload.New(bucket).Execute,
		temporalsdk_activity.RegisterOptions{Name: bucketupload.Name},
	)
	r.RegisterActivityWithOptions(
		removepaths.New().Execute,
		temporalsdk_activity.RegisterOptions{Name: removepaths.Name},
	)
}
