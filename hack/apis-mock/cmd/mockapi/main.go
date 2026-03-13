package main

import (
	"log"
	"net/http"
	"os"

	"github.com/artefactual-sdps/preprocessing-sfa/hack/apis-mock/internal/gen"
	"github.com/artefactual-sdps/preprocessing-sfa/hack/apis-mock/internal/mock"
)

func main() {
	addr := ":" + envOrDefault("PORT", "8080")

	handler := mock.NewHandler()
	security := mock.NewSecurity(envOrDefault("MOCK_AUTH_TOKEN", "mock-token"))
	server, err := gen.NewServer(handler, security)
	if err != nil {
		log.Fatalf("create server: %v", err)
	}

	log.Printf("mock API listening on %s", addr)
	if err := http.ListenAndServe(addr, server); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}

func envOrDefault(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
