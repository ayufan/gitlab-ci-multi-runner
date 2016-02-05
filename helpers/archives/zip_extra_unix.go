// +build linux darwin freebsd

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

func createZipUidGidField(w io.Writer, fi os.FileInfo) (err error) {
	stat, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return
	}

	ugField := ZipUidGidField{
		1,
		4, stat.Uid,
		4, stat.Gid,
	}
	ugFieldType := ZipExtraField{
		Type: ZipUidGidFieldType,
		Size: uint16(binary.Size(&ugField)),
	}
	err = binary.Write(w, binary.LittleEndian, &ugFieldType)
	if err == nil {
		err = binary.Write(w, binary.LittleEndian, &ugField)
	}
	return nil
}

func processZipUidGidField(data []byte, file *zip.FileHeader) error {
	var ugField ZipUidGidField
	err := binary.Read(bytes.NewReader(data), binary.LittleEndian, &ugField)
	if err != nil {
		return err
	}

	if !(ugField.Version == 1 && ugField.UIDSize == 4 && ugField.GIDSize == 4) {
		return errors.New("uid/gid data not supported")
	}

	return os.Lchown(file.Name, int(ugField.Uid), int(ugField.Gid))
}
