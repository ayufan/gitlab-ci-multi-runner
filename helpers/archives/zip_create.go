package archives

import (
	"archive/zip"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/Sirupsen/logrus"
)

func createZipEntry(archive *zip.Writer, fileName string) error {
	fi, err := os.Lstat(fileName)
	if err != nil {
		logrus.Warningln("File ignored:", err)
		return nil
	}

	fh, err := zip.FileInfoHeader(fi)
	fh.Name = fileName
	fh.Extra = createZipExtra(fi)

	switch fi.Mode() & os.ModeType {
	case os.ModeDir:
		fh.Name += "/"

		_, err := archive.CreateHeader(fh)
		if err != nil {
			return err
		}

	case os.ModeSymlink:
		fw, err := archive.CreateHeader(fh)
		if err != nil {
			return err
		}

		link, err := os.Readlink(fileName)
		if err != nil {
			return err
		}

		io.WriteString(fw, link)

	case os.ModeNamedPipe, os.ModeSocket, os.ModeDevice:
		// Ignore the files that of these types
		logrus.Warningln("File ignored:", fileName)

	default:
		fh.Method = zip.Deflate
		fw, err := archive.CreateHeader(fh)
		if err != nil {
			return err
		}

		file, err := os.Open(fileName)
		if err != nil {
			return err
		}

		_, err = io.Copy(fw, file)
		file.Close()
		if err != nil {
			return err
		}
		break
	}
	return nil
}

func CreateZipArchive(w io.Writer, fileNames []string) error {
	archive := zip.NewWriter(w)
	defer archive.Close()

	for _, fileName := range fileNames {
		err := createZipEntry(archive, fileName)
		if err != nil {
			return err
		}
	}

	return nil
}

func CreateZipFile(fileName string, fileNames []string) error {
	// create directories to store archive
	os.MkdirAll(filepath.Dir(fileName), 0700)

	tempFile, err := ioutil.TempFile(filepath.Dir(fileName), "archive_")
	if err != nil {
		return err
	}
	defer tempFile.Close()
	defer os.Remove(tempFile.Name())

	logrus.Debugln("Temporary file:", tempFile.Name())
	err = CreateZipArchive(tempFile, fileNames)
	if err != nil {
		return err
	}
	tempFile.Close()

	err = os.Rename(tempFile.Name(), fileName)
	if err != nil {
		return err
	}

	return nil
}
