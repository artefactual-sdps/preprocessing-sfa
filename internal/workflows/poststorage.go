package workflows

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/artefactual-sdps/enduro/pkg/childwf"
	"github.com/artefactual-sdps/temporal-activities/removepaths"
	temporalsdk_temporal "go.temporal.io/sdk/temporal"
	temporalsdk_workflow "go.temporal.io/sdk/workflow"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/amss"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/apis"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/config"
)

type Poststorage struct {
	cfg config.PoststorageConfig
}

func NewPoststorage(cfg config.PoststorageConfig) *Poststorage {
	return &Poststorage{cfg: cfg}
}

func (w *Poststorage) Execute(
	ctx temporalsdk_workflow.Context,
	params *childwf.PostStorageParams,
) (r *childwf.PostStorageResult, e error) {
	logger := temporalsdk_workflow.GetLogger(ctx)
	logger.Debug("Poststorage workflow running!", "params", params)

	if data, ok := params.CustomMetadata[apis.CustomMetadataKey]; ok {
		var metadata apis.CustomMetadata
		if err := metadata.Unmarshal(data); err != nil {
			return nil, err
		}
		logger.Info(
			"Received APIS custom metadata.",
			"importTaskID", metadata.ImportTaskID,
			"decision", metadata.Decision,
		)
	}

	defer func() {
		logger.Debug("Poststorage workflow finished!", "result", r, "error", e)
	}()

	var getAIPPathResult amss.GetAIPPathActivityResult
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
		amss.GetAIPPathActivityName,
		&amss.GetAIPPathActivityParams{
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
		})
		for attempt := 1; attempt <= maxAttempts; attempt++ {
			sessCtx, err := temporalsdk_workflow.CreateSession(activityOpts, &temporalsdk_workflow.SessionOptions{
				CreationTimeout:  forever,
				ExecutionTimeout: forever,
			})
			if err != nil {
				return nil, fmt.Errorf("error creating session: %v", err)
			}

			sessErr = w.SessionHandler(sessCtx, params.AIPUUID, getAIPPathResult.Path)

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

	return &childwf.PostStorageResult{}, nil
}

func (w *Poststorage) SessionHandler(ctx temporalsdk_workflow.Context, aipUUID, aipPath string) (e error) {
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
	localDir := filepath.Join(w.cfg.WorkingDir, fmt.Sprintf("search-md_%s", aipDirName))
	metsName := fmt.Sprintf("METS.%s.xml", aipUUID)
	metsPath := filepath.Join(localDir, metsName)

	removePaths = append(removePaths, localDir)

	var fetchMETSResult amss.FetchActivityResult
	e = temporalsdk_workflow.ExecuteActivity(
		withActivityOptsForLongLivedRequest(ctx),
		amss.FetchActivityName,
		&amss.FetchActivityParams{
			AIPUUID:      aipUUID,
			RelativePath: fmt.Sprintf("%s/data/%s", aipDirName, metsName),
			Destination:  metsPath,
		},
	).Get(ctx, &fetchMETSResult)
	if e != nil {
		return e
	}

	return nil
}
