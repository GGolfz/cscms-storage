package encrypt

import "io"

type Manager interface {
	Encrypt(input io.Reader) (io.Reader, string, error)
	Decrypt(input io.Reader, nonceString string) (io.Reader, error)
}
