package token

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGenerateFileToken(t *testing.T) {
	nanoIDManager := NewNanoIDTokenManager()
	token, err := nanoIDManager.GenerateFileToken()
	require.NoError(t, err)
	require.Equal(t, len(token), 6)
}

func TestGenerateFileID(t *testing.T) {
	nanoIDManager := NewNanoIDTokenManager()
	token, err := nanoIDManager.GenerateFileID()
	require.NoError(t, err)
	require.Equal(t, len(token), 30)
}

func TestGenerateImageToken(t *testing.T) {
	nanoIDManager := NewNanoIDTokenManager()
	token, err := nanoIDManager.GenerateImageToken()
	require.NoError(t, err)
	require.Equal(t, len(token), 20)
}

func TestGenerateAPIToken(t *testing.T) {
	nanoIDManager := NewNanoIDTokenManager()
	token, err := nanoIDManager.GenerateAPIToken()
	require.NoError(t, err)
	require.Equal(t, len(token), 30)
}
