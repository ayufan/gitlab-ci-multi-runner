package common

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers"
)

type BuildOptions map[string]interface{}

func (m *BuildOptions) MarshalJSON() ([]byte, error) {
	return json.Marshal(*m)
}

func (m *BuildOptions) UnmarshalJSON(data []byte) error {
	if m == nil {
		return errors.New("BuildOptions: UnmarshalJSON on nil pointer")
	}
	return json.Unmarshal(data, *m)
}

func (m *BuildOptions) Get(keys ...string) (interface{}, bool) {
	return helpers.GetMapKey(*m, keys...)
}

func (m *BuildOptions) GetString(keys ...string) (result string, ok bool) {
	value, ok := helpers.GetMapKey(*m, keys...)
	if ok {
		result, ok = value.(string)
	}
	return
}

func (m *BuildOptions) Decode(result interface{}, keys ...string) error {
	value, ok := m.Get(keys...)
	if !ok {
		return fmt.Errorf("key not found %v", strings.Join(keys, "."))
	}

	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, result)
}
