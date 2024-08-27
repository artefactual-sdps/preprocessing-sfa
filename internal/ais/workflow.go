package ais

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/artefactual-sdps/temporal-activities/archivezip"
	"github.com/artefactual-sdps/temporal-activities/bucketupload"
	"github.com/artefactual-sdps/temporal-activities/removepaths"
	"github.com/google/uuid"
	temporalsdk_api_enums "go.temporal.io/api/enums/v1"
	temporalsdk_activity "go.temporal.io/sdk/activity"
	temporalsdk_client "go.temporal.io/sdk/client"
	temporalsdk_temporal "go.temporal.io/sdk/temporal"
	temporalsdk_worker "go.temporal.io/sdk/worker"
	temporalsdk_workflow "go.temporal.io/sdk/workflow"
	"gocloud.dev/blob"
)

// We use this constant to represent a long period of time (10 years).
const forever = time.Hour * 24 * 365 * 10

type WorkflowParams struct {
	AIPUUID uuid.UUID
	AIPName string
}

type WorkflowResult struct {
	Key string
}

type Workflow struct {
	workingDir string
	config     Config
}

func NewWorkflow(workingDir string) *Workflow {
	return &Workflow{workingDir: workingDir}
}

func (w *Workflow) Execute(ctx temporalsdk_workflow.Context, params *WorkflowParams) (r *WorkflowResult, e error) {
	r = &WorkflowResult{}

	logger := temporalsdk_workflow.GetLogger(ctx)
	logger.Debug("AIS workflow running!", "params", params)

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
				return r, fmt.Errorf("error creating session: %v", err)
			}

			r.Key, sessErr = w.SessionHandler(sessCtx, params)

			// We want to retry the session if it has been canceled as a result
			// of losing the worker but not otherwise. This scenario seems to be
			// identifiable when we have an error but the root context has not
			// been canceled.
			if sessErr != nil &&
				(errors.Is(sessErr, temporalsdk_workflow.ErrSessionFailed) || temporalsdk_temporal.IsCanceledError(sessErr)) {
				// Root context canceled, hence workflow canceled.
				if ctx.Err() == temporalsdk_workflow.ErrCanceled {
					return r, nil
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
			return r, sessErr
		}
	}

	logger.Debug("AIS workflow finished!", "result", r, "error", e)

	return r, nil
}

func (w *Workflow) SessionHandler(ctx temporalsdk_workflow.Context, params *WorkflowParams) (s string, e error) {
	removePaths := []string{}

	defer func() {
		var removeResult removepaths.Result
		err := temporalsdk_workflow.ExecuteActivity(
			withLocalActOpts(ctx),
			removepaths.Name,
			&removepaths.Params{Paths: removePaths},
		).Get(ctx, &removeResult)
		if err != nil {
			e = errors.Join(e, err)
		}

		temporalsdk_workflow.CompleteSession(ctx)
	}()

	aipDirName := fmt.Sprintf("%s-%s", params.AIPName, params.AIPUUID.String())
	localDir := filepath.Join(w.workingDir, fmt.Sprintf("search-md_%s", aipDirName))
	metsName := fmt.Sprintf("METS.%s.xml", params.AIPUUID.String())
	metsPath := filepath.Join(localDir, metsName)

	removePaths = append(removePaths, localDir)

	var fetchMETSResult FetchActivityResult
	e = temporalsdk_workflow.ExecuteActivity(
		withRemoteActOpts(ctx),
		FetchActivityName,
		&FetchActivityParams{
			AIPUUID:      params.AIPUUID,
			RelativePath: fmt.Sprintf("%s/data/%s", aipDirName, metsName),
			Destination:  metsPath,
		},
	).Get(ctx, &fetchMETSResult)
	if e != nil {
		return "", e
	}

	var parseResult ParseActivityResult
	e = temporalsdk_workflow.ExecuteActivity(
		withLocalActOpts(ctx),
		ParseActivityName,
		&ParseActivityParams{METSPath: metsPath},
	).Get(ctx, &parseResult)
	if e != nil {
		return "", e
	}

	if parseResult.UpdatedAreldaMetadataRelPath != "" {
		var fetchMetadataResult FetchActivityResult
		e = temporalsdk_workflow.ExecuteActivity(
			withRemoteActOpts(ctx),
			FetchActivityName,
			&FetchActivityParams{
				AIPUUID:      params.AIPUUID,
				RelativePath: fmt.Sprintf("%s/data/%s", aipDirName, parseResult.UpdatedAreldaMetadataRelPath),
				Destination:  filepath.Join(localDir, filepath.Base(parseResult.UpdatedAreldaMetadataRelPath)),
			},
		).Get(ctx, &fetchMetadataResult)
		if e != nil {
			return "", e
		}
	}

	if parseResult.UpdatedAreldaMetadataRelPath == "" && parseResult.MetadataRelPath != "" {
		var fetchMetadataResult FetchActivityResult
		e = temporalsdk_workflow.ExecuteActivity(
			withRemoteActOpts(ctx),
			FetchActivityName,
			&FetchActivityParams{
				AIPUUID:      params.AIPUUID,
				RelativePath: fmt.Sprintf("%s/data/%s", aipDirName, parseResult.MetadataRelPath),
				Destination:  filepath.Join(localDir, filepath.Base(parseResult.MetadataRelPath)),
			},
		).Get(ctx, &fetchMetadataResult)
		if e != nil {
			return "", e
		}
	}

	var zipResult archivezip.Result
	e = temporalsdk_workflow.ExecuteActivity(
		withLocalActOpts(ctx),
		archivezip.Name,
		&archivezip.Params{SourceDir: localDir},
	).Get(ctx, &zipResult)
	if e != nil {
		return "", e
	}

	removePaths = append(removePaths, zipResult.Path)

	var uploadResult bucketupload.Result
	e = temporalsdk_workflow.ExecuteActivity(
		withRemoteActOpts(ctx),
		bucketupload.Name,
		&bucketupload.Params{Path: zipResult.Path},
	).Get(ctx, &uploadResult)
	if e != nil {
		return "", e
	}

	return uploadResult.Key, nil
}

func StartWorkflow(
	ctx context.Context,
	tc temporalsdk_client.Client,
	cfg TemporalConfig,
	params *WorkflowParams,
) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	opts := temporalsdk_client.StartWorkflowOptions{
		ID:                    fmt.Sprintf("%s-%s", cfg.WorkflowName, params.AIPUUID.String()),
		TaskQueue:             cfg.TaskQueue,
		WorkflowIDReusePolicy: temporalsdk_api_enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
	}
	_, err := tc.ExecuteWorkflow(ctx, opts, cfg.WorkflowName, params)

	return err
}

func RegisterWorkflow(ctx context.Context, tw temporalsdk_worker.Worker, config Config, bucket *blob.Bucket) error {
	amssclient, err := NewAMSSClient(config.AMSS)
	if err != nil {
		return fmt.Errorf("RegisterWorkflow: %w", err)
	}

	tw.RegisterWorkflowWithOptions(
		NewWorkflow(config.WorkingDir).Execute,
		temporalsdk_workflow.RegisterOptions{Name: config.Temporal.WorkflowName},
	)
	tw.RegisterActivityWithOptions(
		NewFetchActivity(amssclient).Execute,
		temporalsdk_activity.RegisterOptions{Name: FetchActivityName},
	)
	tw.RegisterActivityWithOptions(
		NewParseActivity().Execute,
		temporalsdk_activity.RegisterOptions{Name: ParseActivityName},
	)
	tw.RegisterActivityWithOptions(
		archivezip.New().Execute,
		temporalsdk_activity.RegisterOptions{Name: archivezip.Name},
	)
	tw.RegisterActivityWithOptions(
		bucketupload.New(bucket).Execute,
		temporalsdk_activity.RegisterOptions{Name: bucketupload.Name},
	)
	tw.RegisterActivityWithOptions(
		removepaths.New().Execute,
		temporalsdk_activity.RegisterOptions{Name: removepaths.Name},
	)

	return nil
}

func withLocalActOpts(ctx temporalsdk_workflow.Context) temporalsdk_workflow.Context {
	return temporalsdk_workflow.WithActivityOptions(ctx, temporalsdk_workflow.ActivityOptions{
		ScheduleToCloseTimeout: 5 * time.Minute,
		RetryPolicy: &temporalsdk_temporal.RetryPolicy{
			MaximumAttempts: 1,
		},
	})
}

func withRemoteActOpts(ctx temporalsdk_workflow.Context) temporalsdk_workflow.Context {
	return temporalsdk_workflow.WithActivityOptions(ctx, temporalsdk_workflow.ActivityOptions{
		HeartbeatTimeout:    time.Minute,
		StartToCloseTimeout: 15 * time.Minute,
		RetryPolicy: &temporalsdk_temporal.RetryPolicy{
			MaximumAttempts: 5,
			InitialInterval: time.Minute,
		},
	})
}
