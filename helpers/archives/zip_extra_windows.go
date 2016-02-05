package archives

import (
	"archive/zip"
	"io"
	"os"
)

func createZipUidGidField(w io.Writer, fi os.FileInfo) (err error) {
	// TODO: currently not supported
	return nil
}

func processZipUidGidField(data []byte, file *zip.FileHeader) error {
	// TODO: currently not supported
	return nil
}
