package archives

import (
	"archive/zip"
	"io"
	"os"
)

func createZipUIDGidField(w io.Writer, fi os.FileInfo) (err error) {
	// TODO: currently not supported
	return nil
}

func processZipUIDGidField(data []byte, file *zip.FileHeader) error {
	// TODO: currently not supported
	return nil
}
