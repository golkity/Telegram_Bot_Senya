package compressor

import (
	"compress/gzip"
	"io"
)

func DecompressStream(src io.Reader) (io.Reader, error) {
	return gzip.NewReader(src)
}
