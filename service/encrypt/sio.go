package encrypt

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"github.com/minio/sio"
	"go.uber.org/zap"
	"golang.org/x/crypto/hkdf"
	"io"
)

type SIOEncryptionManager struct {
	log       *zap.SugaredLogger
	masterKey string
}

func NewSIOEncryptionManager(l *zap.SugaredLogger, key string) *SIOEncryptionManager {
	return &SIOEncryptionManager{
		log:       l,
		masterKey: key,
	}
}

func (m *SIOEncryptionManager) Encrypt(input io.Reader) (io.Reader, string, error) {
	// the master key used to derive encryption keys
	masterKey := []byte(m.masterKey)

	// generate a random nonce to derive an encryption key from the master key
	// this nonce must be saved to be able to decrypt the data again
	var nonce [32]byte
	if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
		m.log.Errorw("Failed to read random data", "error", err)
		return nil, "", err
	}

	// derive an encryption key from the master key and the nonce
	var key [32]byte
	kdf := hkdf.New(sha256.New, masterKey, nonce[:], nil)
	if _, err := io.ReadFull(kdf, key[:]); err != nil {
		m.log.Errorw("Failed to derive encryption key", "error", err)
		return nil, "", err
	}

	encrypted, err := sio.EncryptReader(input, sio.Config{Key: key[:]})
	if err != nil {
		m.log.Errorw("Failed to encrypted reader", "error", err)
		return nil, "", err
	}

	return encrypted, hex.EncodeToString(nonce[:]), nil
}

func (m *SIOEncryptionManager) Decrypt(input io.Reader, nonceString string) (io.Reader, error) {
	// the master key used to derive encryption keys
	masterKey := []byte(m.masterKey)

	// the nonce used to derive the encryption key
	nonce, err := hex.DecodeString(nonceString)
	if err != nil {
		m.log.Errorw("Failed to decode hex string to byte", "error", err)
		return nil, err
	}

	// derive the encryption key from the master key and the nonce
	var key [32]byte
	kdf := hkdf.New(sha256.New, masterKey, nonce, nil)
	if _, err := io.ReadFull(kdf, key[:]); err != nil {
		m.log.Errorw("Failed to derive encryption key", "error", err)
		return nil, err
	}

	return sio.DecryptReader(input, sio.Config{Key: key[:]})
}
