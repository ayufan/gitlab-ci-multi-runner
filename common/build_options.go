package common

import (
	"encoding/json"
	"fmt"
	"strings"

	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers"
)

type BuildOptions map[string]interface{}

func (m *BuildOptions) Get(keys ...string) (interface{}, bool) {
	return helpers.GetMapKey(*m, keys...)
}

func (m *BuildOptions) GetSubOptions(keys ...string) (result BuildOptions, ok bool) {
	value, ok := helpers.GetMapKey(*m, keys...)
	if ok {
		result, ok = value.(map[string]interface{})
	}
	return
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

func convertMapToStringMap(in interface{}) (out interface{}, err error) {
	mapString, ok := in.(map[string]interface{})
	if ok {
		for k, v := range mapString {
			mapString[k], err = convertMapToStringMap(v)
			if err != nil {
				return
			}
		}
		return mapString, nil
	}

	mapInterface, ok := in.(map[interface{}]interface{})
	if ok {
		mapString := make(map[string]interface{})
		for k, v := range mapInterface {
			key, ok := k.(string)
			if !ok {
				return nil, fmt.Errorf("failed to convert %v to string", k)
			}

			mapString[key], err = convertMapToStringMap(v)
			if err != nil {
				return
			}
		}
		return mapString, nil
	}

	return in, nil
}

func (m *BuildOptions) Sanitize() (err error) {
	n := make(BuildOptions)
	for k, v := range *m {
		n[k], err = convertMapToStringMap(v)
		if err != nil {
			return
		}
	}
	*m = n
	return
}
