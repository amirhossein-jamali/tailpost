package security

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/amirhossein-jamali/tailpost/pkg/config"
)

func TestNewEncryption(t *testing.T) {
	// Test with nil config
	encryption, err := NewEncryption(nil)
	if err != nil {
		t.Errorf("Unexpected error with nil config: %v", err)
	}
	if encryption != nil {
		t.Errorf("Expected nil encryption with nil config")
	}

	// Test with disabled encryption
	encConfig := &config.EncryptionConfig{
		Enabled: false,
	}
	encryption, err = NewEncryption(encConfig)
	if err != nil {
		t.Errorf("Unexpected error with disabled encryption: %v", err)
	}
	if encryption != nil {
		t.Errorf("Expected nil encryption with disabled encryption")
	}

	// Test with unsupported algorithm
	encConfig = &config.EncryptionConfig{
		Enabled:   true,
		Algorithm: "unsupported",
		KeyFile:   "testdata/key.txt",
	}
	_, err = NewEncryption(encConfig)
	if err == nil {
		t.Errorf("Expected error with unsupported algorithm")
	}

	// Create temp key file
	keyFile, err := os.CreateTemp("", "key-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(keyFile.Name())

	// Write 32-byte key for AES-GCM
	key := make([]byte, 32)
	for i := 0; i < len(key); i++ {
		key[i] = byte(i)
	}
	if _, err := keyFile.Write(key); err != nil {
		t.Fatalf("Failed to write to key file: %v", err)
	}
	if err := keyFile.Close(); err != nil {
		t.Fatalf("Failed to close key file: %v", err)
	}

	// Test AES-GCM
	encConfig = &config.EncryptionConfig{
		Enabled:   true,
		Algorithm: "aes-gcm",
		KeyFile:   keyFile.Name(),
	}
	encryption, err = NewEncryption(encConfig)
	if err != nil {
		t.Errorf("Unexpected error with AES-GCM: %v", err)
	}
	if encryption == nil {
		t.Errorf("Expected non-nil encryption with AES-GCM")
	}

	// Test ChaCha20-Poly1305
	encConfig = &config.EncryptionConfig{
		Enabled:   true,
		Algorithm: "chacha20-poly1305",
		KeyFile:   keyFile.Name(),
	}
	encryption, err = NewEncryption(encConfig)
	if err != nil {
		t.Errorf("Unexpected error with ChaCha20-Poly1305: %v", err)
	}
	if encryption == nil {
		t.Errorf("Expected non-nil encryption with ChaCha20-Poly1305")
	}

	// Test with non-existent key file
	encConfig = &config.EncryptionConfig{
		Enabled:   true,
		Algorithm: "aes-gcm",
		KeyFile:   "/non/existent/file.txt",
	}
	_, err = NewEncryption(encConfig)
	if err == nil {
		t.Errorf("Expected error with non-existent key file")
	}

	// Create temp key file with wrong key size
	shortKeyFile, err := os.CreateTemp("", "short-key-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(shortKeyFile.Name())

	// Write 16-byte key (too short for our requirements)
	shortKey := make([]byte, 16)
	if _, err := shortKeyFile.Write(shortKey); err != nil {
		t.Fatalf("Failed to write to key file: %v", err)
	}
	if err := shortKeyFile.Close(); err != nil {
		t.Fatalf("Failed to close key file: %v", err)
	}

	// Test with key file that's too short
	encConfig = &config.EncryptionConfig{
		Enabled:   true,
		Algorithm: "aes-gcm",
		KeyFile:   shortKeyFile.Name(),
	}
	_, err = NewEncryption(encConfig)
	if err == nil {
		t.Errorf("Expected error with key file that's too short")
	}
}

func TestEncryptionWithEnvironmentVariable(t *testing.T) {
	// Create a 32-byte key and encode as hex
	key := make([]byte, 32)
	for i := 0; i < len(key); i++ {
		key[i] = byte(i)
	}
	hexKey := "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"

	// Set environment variable
	envVarName := "TEST_ENCRYPTION_KEY"
	os.Setenv(envVarName, hexKey)
	defer os.Unsetenv(envVarName)

	// Test AES-GCM with environment variable
	encConfig := &config.EncryptionConfig{
		Enabled:   true,
		Algorithm: "aes-gcm",
		KeyEnv:    envVarName,
	}

	encryption, err := NewEncryption(encConfig)
	if err != nil {
		t.Errorf("Unexpected error with AES-GCM using env var: %v", err)
	}
	if encryption == nil {
		t.Errorf("Expected non-nil encryption with AES-GCM using env var")
	}

	// Test with invalid hex in environment variable
	os.Setenv(envVarName, "not-hex-data")
	_, err = NewEncryption(encConfig)
	if err == nil {
		t.Errorf("Expected error with invalid hex in environment variable")
	}

	// Test with empty environment variable
	os.Setenv(envVarName, "")
	_, err = NewEncryption(encConfig)
	if err == nil {
		t.Errorf("Expected error with empty environment variable")
	}

	// Test with non-existent environment variable
	encConfig.KeyEnv = "NON_EXISTENT_ENV_VAR"
	_, err = NewEncryption(encConfig)
	if err == nil {
		t.Errorf("Expected error with non-existent environment variable")
	}
}

func TestEncryptionKeyID(t *testing.T) {
	// Create temp key file
	keyFile, err := os.CreateTemp("", "key-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(keyFile.Name())

	// Write 32-byte key
	key := make([]byte, 32)
	if _, err := keyFile.Write(key); err != nil {
		t.Fatalf("Failed to write to key file: %v", err)
	}
	if err := keyFile.Close(); err != nil {
		t.Fatalf("Failed to close key file: %v", err)
	}

	// Test with custom key ID
	customKeyID := "test-key-id-123"
	encConfig := &config.EncryptionConfig{
		Enabled:   true,
		Algorithm: "aes-gcm",
		KeyFile:   keyFile.Name(),
		KeyID:     customKeyID,
	}

	encryption, err := NewEncryption(encConfig)
	if err != nil {
		t.Fatalf("Failed to create encryption: %v", err)
	}

	if encryption.GetKeyID() != customKeyID {
		t.Errorf("Expected key ID %s, got %s", customKeyID, encryption.GetKeyID())
	}

	// Test with auto-generated key ID
	encConfig.KeyID = ""
	encryption, err = NewEncryption(encConfig)
	if err != nil {
		t.Fatalf("Failed to create encryption: %v", err)
	}

	if encryption.GetKeyID() == "" {
		t.Errorf("Expected non-empty auto-generated key ID")
	}
	if !isValidAutoKeyID(encryption.GetKeyID()) {
		t.Errorf("Auto-generated key ID %s doesn't match expected format", encryption.GetKeyID())
	}
}

// Helper function to check if a key ID matches the auto-generated format
func isValidAutoKeyID(keyID string) bool {
	// Auto-generated key IDs are in the format "key-<timestamp>"
	var timestamp int64
	n, err := fmt.Sscanf(keyID, "key-%d", &timestamp)
	return err == nil && n == 1 && timestamp > 0
}

func TestEncryptionRoundTrip(t *testing.T) {
	// Create temp key file
	keyFile, err := os.CreateTemp("", "key-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(keyFile.Name())

	// Write 32-byte key for AES-GCM
	key := make([]byte, 32)
	for i := 0; i < len(key); i++ {
		key[i] = byte(i)
	}
	if _, err := keyFile.Write(key); err != nil {
		t.Fatalf("Failed to write to key file: %v", err)
	}
	if err := keyFile.Close(); err != nil {
		t.Fatalf("Failed to close key file: %v", err)
	}

	testAlgorithms := []string{"aes-gcm", "chacha20-poly1305"}
	testData := []string{
		"",                   // Empty string
		"Hello World",        // Simple string
		"Special !@#$%^&*()", // Special characters
		"Unicode: 你好, world", // Unicode
		"Very long string that is more than a few bytes to ensure we test with data that spans multiple blocks",
	}

	for _, algorithm := range testAlgorithms {
		t.Run(algorithm, func(t *testing.T) {
			encConfig := &config.EncryptionConfig{
				Enabled:   true,
				Algorithm: algorithm,
				KeyFile:   keyFile.Name(),
			}
			encryption, err := NewEncryption(encConfig)
			if err != nil {
				t.Fatalf("Failed to create encryption: %v", err)
			}

			for _, data := range testData {
				// Encrypt
				encrypted, err := encryption.Encrypt([]byte(data))
				if err != nil {
					t.Errorf("Encryption failed for '%s': %v", data, err)
					continue
				}

				// Encrypted data should be different from original
				if len(data) > 0 && string(encrypted) == data {
					t.Errorf("Encrypted data matches original for '%s'", data)
				}

				// Decrypt
				decrypted, err := encryption.Decrypt(encrypted)
				if err != nil {
					t.Errorf("Decryption failed for '%s': %v", data, err)
					continue
				}

				// Check round trip
				if string(decrypted) != data {
					t.Errorf("Round trip failed for '%s', got '%s'", data, string(decrypted))
				}
			}

			// Test tampering with encrypted data
			if len(testData) > 0 {
				data := testData[0]
				encrypted, err := encryption.Encrypt([]byte(data))
				if err != nil {
					t.Fatalf("Encryption failed: %v", err)
				}

				// Tamper with the encrypted data if it's long enough
				if len(encrypted) > 10 {
					// Modify a byte in the middle of the ciphertext (not the nonce)
					encrypted[len(encrypted)/2]++

					// Decryption should fail
					_, err = encryption.Decrypt(encrypted)
					if err == nil {
						t.Errorf("Expected decryption to fail with tampered data")
					}
				}
			}
		})
	}
}

func TestEncryptionProviderCrossTalk(t *testing.T) {
	// Create temp key files
	keyFile1, err := os.CreateTemp("", "key1-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(keyFile1.Name())

	keyFile2, err := os.CreateTemp("", "key2-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(keyFile2.Name())

	// Write different keys to the files
	key1 := make([]byte, 32)
	for i := 0; i < len(key1); i++ {
		key1[i] = byte(i)
	}

	key2 := make([]byte, 32)
	for i := 0; i < len(key2); i++ {
		key2[i] = byte(len(key2) - i - 1)
	}

	if _, err := keyFile1.Write(key1); err != nil {
		t.Fatalf("Failed to write to key file 1: %v", err)
	}
	if err := keyFile1.Close(); err != nil {
		t.Fatalf("Failed to close key file 1: %v", err)
	}

	if _, err := keyFile2.Write(key2); err != nil {
		t.Fatalf("Failed to write to key file 2: %v", err)
	}
	if err := keyFile2.Close(); err != nil {
		t.Fatalf("Failed to close key file 2: %v", err)
	}

	// Create two encryption providers with the same algorithm but different keys
	encConfig1 := &config.EncryptionConfig{
		Enabled:   true,
		Algorithm: "aes-gcm",
		KeyFile:   keyFile1.Name(),
		KeyID:     "key1",
	}

	encConfig2 := &config.EncryptionConfig{
		Enabled:   true,
		Algorithm: "aes-gcm",
		KeyFile:   keyFile2.Name(),
		KeyID:     "key2",
	}

	encryption1, err := NewEncryption(encConfig1)
	if err != nil {
		t.Fatalf("Failed to create encryption 1: %v", err)
	}

	encryption2, err := NewEncryption(encConfig2)
	if err != nil {
		t.Fatalf("Failed to create encryption 2: %v", err)
	}

	// Test data
	data := "This is a secret message"

	// Encrypt with provider 1
	encrypted, err := encryption1.Encrypt([]byte(data))
	if err != nil {
		t.Fatalf("Encryption failed with provider 1: %v", err)
	}

	// Try to decrypt with provider 2 - should fail
	_, err = encryption2.Decrypt(encrypted)
	if err == nil {
		t.Errorf("Expected decryption to fail when using different provider")
	}

	// Decrypt with provider 1 - should succeed
	decrypted, err := encryption1.Decrypt(encrypted)
	if err != nil {
		t.Errorf("Decryption failed with original provider: %v", err)
	}

	if string(decrypted) != data {
		t.Errorf("Expected decrypted data to match original, got %s", string(decrypted))
	}
}

func TestNewEncryptionProvider(t *testing.T) {
	// Create a temporary key file with 32-byte key for tests
	key := make([]byte, 32)
	for i := 0; i < len(key); i++ {
		key[i] = byte(i)
	}

	tempKeyFile, err := os.CreateTemp("", "encryption-key-test-*.key")
	require.NoError(t, err)
	defer os.Remove(tempKeyFile.Name())

	_, err = tempKeyFile.Write(key)
	require.NoError(t, err)
	err = tempKeyFile.Close()
	require.NoError(t, err)

	tests := []struct {
		name        string
		config      config.EncryptionConfig
		envKey      string
		expectError bool
	}{
		{
			name: "AES-GCM with direct key",
			config: config.EncryptionConfig{
				Enabled:   true,
				Type:      "aes",
				Algorithm: "aes-gcm",
				KeyID:     "test-key-1",
				KeyFile:   tempKeyFile.Name(), // Add key file as source
			},
			expectError: false,
		},
		{
			name: "ChaCha20-Poly1305 with direct key",
			config: config.EncryptionConfig{
				Enabled:   true,
				Type:      "chacha20poly1305",
				Algorithm: "chacha20-poly1305",
				KeyID:     "test-key-2",
				KeyFile:   tempKeyFile.Name(), // Add key file as source
			},
			expectError: false,
		},
		{
			name: "Invalid algorithm",
			config: config.EncryptionConfig{
				Enabled:   true,
				Type:      "unknown",
				Algorithm: "unknown",
				KeyID:     "test-key-3",
				KeyFile:   tempKeyFile.Name(), // Add key file as source
			},
			expectError: true,
		},
		{
			name: "Environment variable key",
			config: config.EncryptionConfig{
				Enabled:   true,
				Type:      "aes",
				Algorithm: "aes-gcm",
				KeyEnv:    "TEST_ENCRYPTION_KEY",
				KeyID:     "test-key-4",
			},
			envKey:      "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f", // 32-byte hex key
			expectError: false,
		},
		{
			name: "Missing environment variable key",
			config: config.EncryptionConfig{
				Enabled:   true,
				Type:      "aes",
				Algorithm: "aes-gcm",
				KeyEnv:    "NONEXISTENT_KEY",
				KeyID:     "test-key-5",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable if needed
			if tt.envKey != "" {
				os.Setenv(tt.config.KeyEnv, tt.envKey)
				defer os.Unsetenv(tt.config.KeyEnv)
			}

			provider, err := NewEncryptionProvider(tt.config)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, provider)

				// Test encryption and decryption
				original := "sensitive data to encrypt"
				encrypted, err := provider.Encrypt([]byte(original))
				assert.NoError(t, err)
				assert.NotEqual(t, original, string(encrypted))

				decrypted, err := provider.Decrypt(encrypted)
				assert.NoError(t, err)
				assert.Equal(t, original, string(decrypted))
			}
		})
	}
}

func TestEncryptionKeyFile(t *testing.T) {
	// Create a temporary key file with 32-byte key
	key := make([]byte, 32)
	for i := 0; i < len(key); i++ {
		key[i] = byte(i)
	}

	keyFile, err := os.CreateTemp("", "encryption-key-test")
	require.NoError(t, err)
	defer os.Remove(keyFile.Name())

	_, err = keyFile.Write(key)
	require.NoError(t, err)
	err = keyFile.Close()
	require.NoError(t, err)

	config := config.EncryptionConfig{
		Enabled:   true,
		Type:      "aes",
		Algorithm: "aes-gcm",
		KeyFile:   keyFile.Name(),
		KeyID:     "file-key-id",
	}

	provider, err := NewEncryptionProvider(config)
	require.NoError(t, err)

	// Test encryption and decryption
	original := "data from file-based key"
	encrypted, err := provider.Encrypt([]byte(original))
	assert.NoError(t, err)

	decrypted, err := provider.Decrypt(encrypted)
	assert.NoError(t, err)
	assert.Equal(t, original, string(decrypted))
}

func TestEncryptionWithAutoKeyID(t *testing.T) {
	// Create a temp key file
	keyFile, err := os.CreateTemp("", "key-auto-id-*.txt")
	require.NoError(t, err)
	defer os.Remove(keyFile.Name())

	// Write 32-byte key
	key := make([]byte, 32)
	for i := 0; i < len(key); i++ {
		key[i] = byte(i)
	}
	_, err = keyFile.Write(key)
	require.NoError(t, err)
	keyFile.Close()

	config := config.EncryptionConfig{
		Enabled:   true,
		Type:      "aes",
		Algorithm: "aes-gcm",
		KeyFile:   keyFile.Name(),
		// No KeyID specified, should auto-generate
	}

	provider, err := NewEncryptionProvider(config)
	require.NoError(t, err)

	// Verify a key ID was generated
	assert.True(t, len(provider.GetKeyID()) > 0)

	// Test encryption and decryption
	original := "data with auto key ID"
	encrypted, err := provider.Encrypt([]byte(original))
	assert.NoError(t, err)

	decrypted, err := provider.Decrypt(encrypted)
	assert.NoError(t, err)
	assert.Equal(t, original, string(decrypted))
}

func TestCrossTalkBetweenProviders(t *testing.T) {
	// Create a temp key file
	keyFile, err := os.CreateTemp("", "key-cross-talk-*.txt")
	require.NoError(t, err)
	defer os.Remove(keyFile.Name())

	// Write 32-byte key
	key := make([]byte, 32)
	for i := 0; i < len(key); i++ {
		key[i] = byte(i)
	}
	_, err = keyFile.Write(key)
	require.NoError(t, err)
	keyFile.Close()

	config1 := config.EncryptionConfig{
		Enabled: true,
		Type:    "aes",
		KeyFile: keyFile.Name(),
		KeyID:   "provider-1",
	}

	config2 := config.EncryptionConfig{
		Enabled: true,
		Type:    "aes",
		KeyFile: keyFile.Name(),
		KeyID:   "provider-2",
	}

	provider1, err := NewEncryptionProvider(config1)
	require.NoError(t, err)

	provider2, err := NewEncryptionProvider(config2)
	require.NoError(t, err)

	// Encrypt with provider1
	original := "cross talk test data"
	encrypted, err := provider1.Encrypt([]byte(original))
	require.NoError(t, err)

	// Try to decrypt with provider2 (should fail)
	_, err = provider2.Decrypt(encrypted)
	assert.Error(t, err)

	// Decrypt with provider1 (should succeed)
	decrypted, err := provider1.Decrypt(encrypted)
	assert.NoError(t, err)
	assert.Equal(t, original, string(decrypted))
}

func TestEncryptionDisabled(t *testing.T) {
	config := config.EncryptionConfig{
		Enabled: false,
	}

	provider, err := NewEncryptionProvider(config)
	require.NoError(t, err)

	// Provider should be nil when encryption is disabled
	assert.Nil(t, provider, "Provider should be nil when encryption is disabled")

	// Now let's try to create a NoOpEncryptionProvider explicitly for testing
	noopProvider := &NoOpEncryptionProvider{keyID: "no-encryption"}

	// With a NoOp provider, data should pass through unchanged
	original := "unencrypted data"
	encrypted, err := noopProvider.Encrypt([]byte(original))
	assert.NoError(t, err)
	assert.Equal(t, original, string(encrypted))

	decrypted, err := noopProvider.Decrypt(encrypted)
	assert.NoError(t, err)
	assert.Equal(t, original, string(decrypted))

	// Verify KeyID functions correctly
	assert.Equal(t, "no-encryption", noopProvider.GetKeyID())
}

// NoOpEncryptionProvider implements the EncryptionProvider interface but doesn't perform any encryption
type NoOpEncryptionProvider struct {
	keyID string
}

// Encrypt for NoOp provider just returns the plaintext unchanged
func (p *NoOpEncryptionProvider) Encrypt(plaintext []byte) ([]byte, error) {
	return plaintext, nil
}

// Decrypt for NoOp provider just returns the ciphertext unchanged
func (p *NoOpEncryptionProvider) Decrypt(ciphertext []byte) ([]byte, error) {
	return ciphertext, nil
}

// GetKeyID returns the key ID
func (p *NoOpEncryptionProvider) GetKeyID() string {
	return p.keyID
}
