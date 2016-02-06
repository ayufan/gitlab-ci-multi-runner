package archives

import (
	"archive/zip"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/Sirupsen/logrus"
)

func createZipDirectoryEntry(archive *zip.Writer, fh *zip.FileHeader) error {
	fh.Name += "/"
	_, err := archive.CreateHeader(fh)
	return err
}

func createZipSymlinkEntry(archive *zip.Writer, fh *zip.FileHeader) error {
	fw, err := archive.CreateHeader(fh)
	if err != nil {
		return err
	}

	link, err := os.Readlink(fh.Name)
	if err != nil {
		return err
	}

	_, err = io.WriteString(fw, link)
	return err
}

func createZipFileEntry(archive *zip.Writer, fh *zip.FileHeader) error {
	fh.Method = zip.Deflate
	fw, err := archive.CreateHeader(fh)
	if err != nil {
		return err
	}

	file, err := os.Open(fh.Name)
	if err != nil {
		return err
	}

	_, err = io.Copy(fw, file)
	file.Close()
	if err != nil {
		return err
	}
	return nil
}

func createZipEntry(archive *zip.Writer, fileName string) error {
	fi, err := os.Lstat(fileName)
	if err != nil {
		logrus.Warningln("File ignored:", err)
		return nil
	}

	fh, err := zip.FileInfoHeader(fi)
	if err != nil {
		return err
	}
	fh.Name = fileName
	fh.Extra = createZipExtra(fi)

	switch fi.Mode() & os.ModeType {
	case os.ModeDir:
		return createZipDirectoryEntry(archive, fh)

	case os.ModeSymlink:
		return createZipSymlinkEntry(archive, fh)

	case os.ModeNamedPipe, os.ModeSocket, os.ModeDevice:
		// Ignore the files that of these types
		logrus.Warningln("File ignored:", fileName)
		return nil

	default:
		return createZipFileEntry(archive, fh)
	}
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
