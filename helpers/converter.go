package helpers

import (
	"bufio"
	"bytes"
	"encoding/json"
	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v1"
)

func ToJson(src interface{}) string {
	data, err := json.Marshal(src)
	if err == nil {
		return string(data)
	} else {
		return ""
	}
}

func ToYAML(src interface{}) string {
	data, err := yaml.Marshal(src)
	if err == nil {
		return string(data)
	} else {
		return ""
	}
}

func ToTOML(src interface{}) string {
	var data bytes.Buffer
	buffer := bufio.NewWriter(&data)

	if err := toml.NewEncoder(buffer).Encode(src); err != nil {
		return ""
	}

	if err := buffer.Flush(); err != nil {
		return ""
	}

	return data.String()
}
