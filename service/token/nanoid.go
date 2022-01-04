package token

import (
	"github.com/matoous/go-nanoid/v2"
)

type NanoIDTokenManager struct{}

func NewNanoIDTokenManager() *NanoIDTokenManager {
	return &NanoIDTokenManager{}
}

func (n *NanoIDTokenManager) GenerateFileToken() (string, error) {
	return gonanoid.Generate("0123456789abcdefghijklmnopqrstuvwxyz", 6)
}

func (n *NanoIDTokenManager) GenerateFileID() (string, error) {
	return gonanoid.New(30)
}

func (n *NanoIDTokenManager) GenerateImageToken() (string, error) {
	return gonanoid.Generate("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz", 20)
}

func (n *NanoIDTokenManager) GenerateAPIToken() (string, error) {
	return gonanoid.New(30)
}
