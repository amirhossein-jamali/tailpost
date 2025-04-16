package main

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

// AESGCMDecrypter for decrypting AES-GCM encrypted data
type AESGCMDecrypter struct {
	aead cipher.AEAD
}

// NewAESGCMDecrypter creates a new AES-GCM decrypter
func NewAESGCMDecrypter(keyHex string) (*AESGCMDecrypter, error) {
	key, err := hex.DecodeString(keyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid hex key: %v", err)
	}

	if len(key) != 32 {
		return nil, fmt.Errorf("AES-256-GCM requires a 32-byte key")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("error creating AES cipher: %v", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("error creating GCM: %v", err)
	}

	return &AESGCMDecrypter{
		aead: aead,
	}, nil
}

// Decrypt decrypts data using AES-GCM
func (d *AESGCMDecrypter) Decrypt(ciphertext []byte, keyID string) ([]byte, error) {
	if len(ciphertext) < d.aead.NonceSize() {
		return nil, fmt.Errorf("ciphertext too short")
	}

	// Split nonce and ciphertext
	nonce, ciphertext := ciphertext[:d.aead.NonceSize()], ciphertext[d.aead.NonceSize():]

	// Decrypt and authenticate
	plaintext, err := d.aead.Open(nil, nonce, ciphertext, []byte(keyID))
	if err != nil {
		return nil, fmt.Errorf("error decrypting: %v", err)
	}

	return plaintext, nil
}

func main() {
	// Get encryption key from environment variable or use a default
	encKey := os.Getenv("TAILPOST_ENCRYPTION_KEY")
	if encKey == "" {
		// Default key for testing
		encKey = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	}

	// Create decrypter
	decrypter, err := NewAESGCMDecrypter(encKey)
	if err != nil {
		log.Fatalf("Failed to create decrypter: %v", err)
	}

	// Handle logs endpoint
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

		// Check if the request is encrypted
		isEncrypted := r.Header.Get("X-Encrypted") == "true"
		keyID := r.Header.Get("X-Key-ID")
		contentType := r.Header.Get("Content-Type")

		var jsonData []byte
		if isEncrypted && contentType == "application/octet-stream" && keyID != "" {
			// Decrypt the data
			jsonData, err = decrypter.Decrypt(body, keyID)
			if err != nil {
				log.Printf("Decryption error: %v", err)
				http.Error(w, "Failed to decrypt data", http.StatusBadRequest)
				return
			}
			log.Printf("Successfully decrypted data with key ID: %s", keyID)
		} else {
			// Not encrypted, use as is
			jsonData = body
		}

		// Parse the JSON logs
		var logs []string
		if err := json.Unmarshal(jsonData, &logs); err != nil {
			log.Printf("JSON error: %v", err)
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
	fmt.Printf("Mock server with decryption listening on %s\n", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
