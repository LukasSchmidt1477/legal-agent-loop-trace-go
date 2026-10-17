package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/example/legal-agent-loop-trace-go/internal/infrai"
	"github.com/example/legal-agent-loop-trace-go/internal/legalflow"
)

func main() {
	apiKey := os.Getenv("INFRAI_API_KEY")
	if apiKey == "" {
		log.Fatal("INFRAI_API_KEY is required")
	}

	client := infrai.NewClient(apiKey)
	service := legalflow.NewMatterIntakeService(client, time.Now)

	mux := http.NewServeMux()
	mux.HandleFunc("/matters", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var input legalflow.MatterIntakeInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}

		result, err := service.ProcessMatter(r.Context(), input)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(result)
	})

	server := &http.Server{
		Addr:              ":8080",
		Handler:           logRequests(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Println("listening on :8080")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), requestKey{}, time.Now().UTC().Format(time.RFC3339))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type requestKey struct{}
