package archives

import (
	"archive/zip"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/Sirupsen/logrus"
)

func extractZipFile(file *zip.File) (err error) {
	fi := file.FileInfo()

	// Create all parents to extract the file
	os.MkdirAll(filepath.Dir(file.Name), 0777)

	switch file.Mode() & os.ModeType {
	case os.ModeDir:
		err = os.Mkdir(file.Name, fi.Mode().Perm())

		// The error that directory does exists is not a error for us
		if os.IsExist(err) {
			err = nil
		}

	case os.ModeSymlink:
		var data []byte
		in, err := file.Open()
		if err == nil {
			defer in.Close()
			data, err = ioutil.ReadAll(in)
		}
		if err == nil {
			// Remove symlink before creating a new one, otherwise we can error that file does exist
			os.Remove(file.Name)
			err = os.Symlink(string(data), file.Name)
		}

	case os.ModeNamedPipe, os.ModeSocket, os.ModeDevice:
		// Ignore the files that of these types
		logrus.Warningln("File ignored: %q", file.Name)

	default:
		var out *os.File
		in, err := file.Open()
		if err == nil {
			defer in.Close()
			// Remove file before creating a new one, otherwise we can error that file does exist
			os.Remove(file.Name)
			out, err = os.OpenFile(file.Name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fi.Mode().Perm())
		}
		if err == nil {
			defer out.Close()
			_, err = io.Copy(out, in)
			out.Close()
		}
		break
	}
	return
}

func ExtractZipArchive(archive *zip.Reader) error {
	for _, file := range archive.File {
		if err := extractZipFile(file); err != nil {
			logrus.Warningf("%s: %s", file.Name, err)
		}
	}

	for _, file := range archive.File {
		// Update file permissions
		if err := os.Chmod(file.Name, file.Mode().Perm()); err != nil {
			logrus.Warningf("%s: %s", file.Name, err)
		}

		// Process zip metadata
		if err := processZipExtra(&file.FileHeader); err != nil {
			logrus.Warningf("%s: %s", file.Name, err)
		}
	}
	return nil
}

func ExtractZipFile(fileName string) error {
	archive, err := zip.OpenReader(fileName)
	if err != nil {
		return err
	}
	defer archive.Close()

	return ExtractZipArchive(&archive.Reader)
}
