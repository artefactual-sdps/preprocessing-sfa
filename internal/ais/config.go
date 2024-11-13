package ais

import "go.artefactual.dev/tools/bucket"

type Config struct {
	WorkingDir string
	Temporal   TemporalConfig
	Worker     WorkerConfig
	AMSS       AMSSConfig
	Bucket     bucket.Config
}

type TemporalConfig struct {
	Address      string
	Namespace    string
	TaskQueue    string
	WorkflowName string
}

type WorkerConfig struct {
	MaxConcurrentSessions int
}

type AMSSConfig struct {
	URL  string
	User string
	Key  string
}
