package ais

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	temporalsdk_client "go.temporal.io/sdk/client"
)

func NewAPIServer(ctx context.Context, tc temporalsdk_client.Client, cfg Config) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/stored", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
			return
		}

		type data struct {
			UUID string `json:"uuid"`
			Name string `json:"name"`
		}
		var payload data
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if payload.Name == "" {
			http.Error(w, "Invalid AIP name", http.StatusBadRequest)
			return
		}

		aipUUID, err := uuid.Parse(payload.UUID)
		if err != nil {
			http.Error(w, "Invalid AIP UUID", http.StatusBadRequest)
			return
		}

		err = StartWorkflow(ctx, tc, cfg.Temporal, &WorkflowParams{
			AIPUUID: aipUUID,
			AIPName: payload.Name,
		})
		if err != nil {
			http.Error(w, "Couldn't start AIS workflow", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusAccepted)
	})

	return &http.Server{
		Addr:         cfg.Listen,
		ReadTimeout:  time.Second * 1,
		WriteTimeout: time.Second * 1,
		IdleTimeout:  time.Second * 30,
		Handler:      mux,
	}
}
