package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	deliveryerrors "github.com/infrai-examples/course-delivery-error-groups"
)

func main() {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		log.Fatal("INFRAI_API_KEY is required")
	}
	client := deliveryerrors.NewClient(key)

	http.HandleFunc("/delivery-errors", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var failure deliveryerrors.DeliveryFailure
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&failure); err != nil {
			http.Error(w, "invalid delivery failure", http.StatusBadRequest)
			return
		}
		decision, err := deliveryerrors.Classify(failure, time.Now().UTC())
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		data, err := client.Capture(r.Context(), decision)
		if err != nil {
			var apiErr *deliveryerrors.APIError
			if errors.As(err, &apiErr) && apiErr.StatusCode >= 400 && apiErr.StatusCode < 500 {
				http.Error(w, apiErr.Error(), apiErr.StatusCode)
				return
			}
			http.Error(w, "upstream request failed", http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]any{"classification": decision.Level, "capture": data})
	})

	log.Println("course error service listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
