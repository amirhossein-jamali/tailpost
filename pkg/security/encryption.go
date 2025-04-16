package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/amirhossein-jamali/tailpost/pkg/config"
	"golang.org/x/crypto/chacha20poly1305"
)

// EncryptionProvider is an interface for data encryption/decryption
type EncryptionProvider interface {
	// Encrypt encrypts the provided plaintext
	Encrypt(plaintext []byte) ([]byte, error)
	// Decrypt decrypts the provided ciphertext
	Decrypt(ciphertext []byte) ([]byte, error)
	// GetKeyID returns the current encryption key ID
	GetKeyID() string
}

// AESGCMProvider implements AES-GCM encryption
type AESGCMProvider struct {
	aead  cipher.AEAD
	keyID string
}

// NewAESGCMProvider creates a new AES-GCM encryption provider
func NewAESGCMProvider(key []byte, keyID string) (*AESGCMProvider, error) {
	if len(key) != 32 {
		return nil, errors.New("AES-256-GCM requires a 32-byte key")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("error creating AES cipher: %v", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("error creating GCM: %v", err)
	}

	return &AESGCMProvider{
		aead:  aead,
		keyID: keyID,
	}, nil
}

// Encrypt encrypts data using AES-GCM
func (p *AESGCMProvider) Encrypt(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, p.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("error generating nonce: %v", err)
	}

	// Prepend keyID (hex encoded) to ciphertext
	ciphertext := p.aead.Seal(nonce, nonce, plaintext, []byte(p.keyID))
	// Format: nonce + ciphertext
	return ciphertext, nil
}

// Decrypt decrypts data using AES-GCM
func (p *AESGCMProvider) Decrypt(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < p.aead.NonceSize() {
		return nil, errors.New("ciphertext too short")
	}

	// Split nonce and ciphertext
	nonce, ciphertext := ciphertext[:p.aead.NonceSize()], ciphertext[p.aead.NonceSize():]

	// Decrypt and authenticate
	plaintext, err := p.aead.Open(nil, nonce, ciphertext, []byte(p.keyID))
	if err != nil {
		return nil, fmt.Errorf("error decrypting: %v", err)
	}

	return plaintext, nil
}

// GetKeyID returns the current encryption key ID
func (p *AESGCMProvider) GetKeyID() string {
	return p.keyID
}

// ChaCha20Poly1305Provider implements ChaCha20-Poly1305 encryption
type ChaCha20Poly1305Provider struct {
	aead  cipher.AEAD
	keyID string
}

// NewChaCha20Poly1305Provider creates a new ChaCha20-Poly1305 encryption provider
func NewChaCha20Poly1305Provider(key []byte, keyID string) (*ChaCha20Poly1305Provider, error) {
	if len(key) != chacha20poly1305.KeySize {
		return nil, fmt.Errorf("ChaCha20-Poly1305 requires a %d-byte key", chacha20poly1305.KeySize)
	}

	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, fmt.Errorf("error creating ChaCha20-Poly1305: %v", err)
	}

	return &ChaCha20Poly1305Provider{
		aead:  aead,
		keyID: keyID,
	}, nil
}

// Encrypt encrypts data using ChaCha20-Poly1305
func (p *ChaCha20Poly1305Provider) Encrypt(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, p.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("error generating nonce: %v", err)
	}

	// Format: nonce + ciphertext
	ciphertext := p.aead.Seal(nonce, nonce, plaintext, []byte(p.keyID))
	return ciphertext, nil
}

// Decrypt decrypts data using ChaCha20-Poly1305
func (p *ChaCha20Poly1305Provider) Decrypt(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < p.aead.NonceSize() {
		return nil, errors.New("ciphertext too short")
	}

	// Split nonce and ciphertext
	nonce, ciphertext := ciphertext[:p.aead.NonceSize()], ciphertext[p.aead.NonceSize():]

	// Decrypt and authenticate
	plaintext, err := p.aead.Open(nil, nonce, ciphertext, []byte(p.keyID))
	if err != nil {
		return nil, fmt.Errorf("error decrypting: %v", err)
	}

	return plaintext, nil
}

// GetKeyID returns the current encryption key ID
func (p *ChaCha20Poly1305Provider) GetKeyID() string {
	return p.keyID
}

// generateKeyID generates a key ID based on timestamp
func generateKeyID() string {
	return fmt.Sprintf("key-%d", time.Now().Unix())
}

// loadKey loads encryption key from file or environment variable
func loadKey(config config.EncryptionConfig) ([]byte, string, error) {
	var key []byte
	var err error
	var keyID string

	// Try to get key from file
	if config.KeyFile != "" {
		key, err = os.ReadFile(config.KeyFile)
		if err != nil {
			return nil, "", fmt.Errorf("error reading key file: %v", err)
		}
	} else if config.KeyEnv != "" {
		// Try to get key from environment variable
		keyStr := os.Getenv(config.KeyEnv)
		if keyStr == "" {
			return nil, "", fmt.Errorf("encryption key environment variable %s is empty", config.KeyEnv)
		}
		key, err = hex.DecodeString(keyStr)
		if err != nil {
			return nil, "", fmt.Errorf("error decoding hex key: %v", err)
		}
	} else {
		return nil, "", errors.New("no key source specified")
	}

	if config.KeyID != "" {
		keyID = config.KeyID
	} else {
		keyID = generateKeyID()
	}

	return key, keyID, nil
}

// NewEncryptionProvider creates a new encryption provider based on configuration
func NewEncryptionProvider(encConfig config.EncryptionConfig) (EncryptionProvider, error) {
	if !encConfig.Enabled {
		return nil, nil
	}

	key, keyID, err := loadKey(encConfig)
	if err != nil {
		return nil, err
	}

	switch encConfig.Type {
	case "aes":
		return NewAESGCMProvider(key, keyID)
	case "chacha20poly1305":
		return NewChaCha20Poly1305Provider(key, keyID)
	default:
		return nil, fmt.Errorf("unsupported encryption type: %s", encConfig.Type)
	}
}

// NewEncryption creates a new encryption provider based on configuration
// This is a wrapper around NewEncryptionProvider for backwards compatibility
func NewEncryption(encConfig *config.EncryptionConfig) (EncryptionProvider, error) {
	if encConfig == nil || !encConfig.Enabled {
		return nil, nil
	}

	// Convert algorithm name to type for compatibility
	config := config.EncryptionConfig{
		Enabled: encConfig.Enabled,
		KeyFile: encConfig.KeyFile,
		KeyEnv:  encConfig.KeyEnv,
		KeyID:   encConfig.KeyID,
	}

	// Map algorithm names to encryption types
	switch encConfig.Algorithm {
	case "aes-gcm":
		config.Type = "aes"
	case "chacha20-poly1305":
		config.Type = "chacha20poly1305"
	default:
		return nil, fmt.Errorf("unsupported encryption algorithm: %s", encConfig.Algorithm)
	}

	return NewEncryptionProvider(config)
}
