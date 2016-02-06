package archives

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"io"
	"os"
	"time"
)

const ZipUIDGidFieldType = 0x7875
const ZipTimestampFieldType = 0x5455

// ZipExtraField is taken from https://github.com/LuaDist/zip/blob/master/proginfo/extrafld.txt
type ZipExtraField struct {
	Type uint16
	Size uint16
}

type ZipUIDGidField struct {
	Version uint8
	UIDSize uint8
	UID     uint32
	GIDSize uint8
	Gid     uint32
}

type ZipTimestampField struct {
	Flags   uint8
	ModTime uint32
}

func createZipTimestampField(w io.Writer, fi os.FileInfo) (err error) {
	tsField := ZipTimestampField{
		1,
		uint32(fi.ModTime().Unix()),
	}
	tsFieldType := ZipExtraField{
		Type: ZipTimestampFieldType,
		Size: uint16(binary.Size(&tsField)),
	}
	err = binary.Write(w, binary.LittleEndian, &tsFieldType)
	if err == nil {
		err = binary.Write(w, binary.LittleEndian, &tsField)
	}
	return
}

func processZipTimestampField(data []byte, file *zip.FileHeader) error {
	if !file.Mode().IsDir() && !file.Mode().IsRegular() {
		return nil
	}

	var tsField ZipTimestampField
	err := binary.Read(bytes.NewReader(data), binary.LittleEndian, &tsField)
	if err != nil {
		return err
	}

	if (tsField.Flags & 1) == 1 {
		modTime := time.Unix(int64(tsField.ModTime), 0)
		acTime := time.Now()
		return os.Chtimes(file.Name, acTime, modTime)
	}

	return nil
}

func createZipExtra(fi os.FileInfo) []byte {
	var buffer bytes.Buffer
	err := createZipUIDGidField(&buffer, fi)
	if err == nil {
		err = createZipTimestampField(&buffer, fi)
	}
	if err == nil {
		return buffer.Bytes()
	}
	return nil
}

func readZipExtraField(r io.Reader) (field ZipExtraField, data []byte, err error) {
	err = binary.Read(r, binary.LittleEndian, &field)
	if err != nil {
		return
	}

	data = make([]byte, field.Size)
	_, err = r.Read(data)
	if err != nil {
		return
	}
	return
}

func processZipExtra(file *zip.FileHeader) error {
	if len(file.Extra) == 0 {
		return nil
	}

	r := bytes.NewReader(file.Extra)
	for {
		field, data, err := readZipExtraField(r)
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		switch field.Type {
		case ZipUIDGidFieldType:
			err = processZipUIDGidField(data, file)
		case ZipTimestampFieldType:
			err = processZipTimestampField(data, file)
		}
		if err != nil {
			return err
		}
	}

	return nil
}
