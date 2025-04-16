package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/logs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}

		var logs []string
		if err := json.Unmarshal(body, &logs); err != nil {
			http.Error(w, "Failed to parse JSON", http.StatusBadRequest)
			return
		}

		fmt.Println("Received logs:")
		for i, logLine := range logs {
			fmt.Printf("[%d] %s\n", i+1, logLine)
		}

		w.WriteHeader(http.StatusOK)
	})

	// Add health endpoint for health checks
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	addr := ":8081"
	fmt.Printf("Mock server listening on %s\n", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
