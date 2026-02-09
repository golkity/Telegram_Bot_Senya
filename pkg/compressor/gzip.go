package compressor

import (
	"compress/gzip"
	"io"
)

func CompressStream(src io.Reader) io.Reader {
	pr, pw := io.Pipe()

	go func() {
		gw := gzip.NewWriter(pw)
		_, err := io.Copy(gw, src)
		if err != nil {
			pw.CloseWithError(err)
			return
		}
		gw.Close()
		pw.Close()
	}()

	return pr
}
