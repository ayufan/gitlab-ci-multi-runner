// +build linux darwin freebsd openbsd

package archives

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"syscall"
)

func createZipUIDGidField(w io.Writer, fi os.FileInfo) (err error) {
	stat, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return
	}

	ugField := ZipUIDGidField{
		1,
		4, stat.Uid,
		4, stat.Gid,
	}
	ugFieldType := ZipExtraField{
		Type: ZipUIDGidFieldType,
		Size: uint16(binary.Size(&ugField)),
	}
	err = binary.Write(w, binary.LittleEndian, &ugFieldType)
	if err == nil {
		err = binary.Write(w, binary.LittleEndian, &ugField)
	}
	return nil
}

func processZipUIDGidField(data []byte, file *zip.FileHeader) error {
	var ugField ZipUIDGidField
	err := binary.Read(bytes.NewReader(data), binary.LittleEndian, &ugField)
	if err != nil {
		return err
	}

	if !(ugField.Version == 1 && ugField.UIDSize == 4 && ugField.GIDSize == 4) {
		return errors.New("uid/gid data not supported")
	}

	return os.Lchown(file.Name, int(ugField.UID), int(ugField.Gid))
}
