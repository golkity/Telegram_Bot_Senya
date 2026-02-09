package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
)

type ConcatReader struct {
	data []byte
	pos  int
}

func EncryptStream(src io.Reader, key []byte) (io.Reader, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	iv := make([]byte, aes.BlockSize)
	if _, err := rand.Read(iv); err != nil {
		return nil, err
	}

	stream := cipher.NewCTR(block, iv)
	encryptedReader := &cipher.StreamReader{S: stream, R: src}

	return io.MultiReader(
		NewConcatReader(iv, nil),
		encryptedReader,
	), nil
}

func NewConcatReader(data []byte, _ interface{}) *ConcatReader {
	return &ConcatReader{
		data: data,
	}
}

func (r *ConcatReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}
