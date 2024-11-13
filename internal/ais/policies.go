package ais

import (
	"time"

	temporalsdk_temporal "go.temporal.io/sdk/temporal"
	temporalsdk_workflow "go.temporal.io/sdk/workflow"
)

// We use this constant to represent a long period of time (10 years).
const forever = time.Hour * 24 * 365 * 10

// withActivityOptsForLongLivedRequest returns a workflow context with activity
// options suited for long-running activities without heartbeats
func withActivityOptsForLongLivedRequest(ctx temporalsdk_workflow.Context) temporalsdk_workflow.Context {
	return temporalsdk_workflow.WithActivityOptions(ctx, temporalsdk_workflow.ActivityOptions{
		StartToCloseTimeout: time.Hour * 2,
		RetryPolicy: &temporalsdk_temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2,
			MaximumInterval:    time.Minute * 10,
			MaximumAttempts:    5,
			NonRetryableErrorTypes: []string{
				"TemporalTimeout:StartToClose",
			},
		},
	})
}

// withFilesystemActivityOpts returns a workflow context with activity
// options suited for activities like disk operations that should not
// require a retry policy attached.
func withFilesystemActivityOpts(ctx temporalsdk_workflow.Context) temporalsdk_workflow.Context {
	return temporalsdk_workflow.WithActivityOptions(ctx, temporalsdk_workflow.ActivityOptions{
		StartToCloseTimeout: time.Hour,
		RetryPolicy: &temporalsdk_temporal.RetryPolicy{
			MaximumAttempts: 1,
		},
	})
}

// withLocalActivityOpts returns a workflow context with activity options suited
// for local and short-lived activities with a few retries.
func withLocalActivityOpts(ctx temporalsdk_workflow.Context) temporalsdk_workflow.Context {
	return temporalsdk_workflow.WithLocalActivityOptions(ctx, temporalsdk_workflow.LocalActivityOptions{
		ScheduleToCloseTimeout: 5 * time.Second,
		RetryPolicy: &temporalsdk_temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
		},
	})
}
