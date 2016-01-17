package commands_helpers

import (
	"archive/zip"
	"io"
	"os"
)

func createZipUidGidField(w io.Writer, fi os.FileInfo) (err error) {
	// TODO: currently not supported
	return nil
}

func processZipUidGidField(data []byte, file *zip.File) error {
	// TODO: currently not supported
	return nil
}
