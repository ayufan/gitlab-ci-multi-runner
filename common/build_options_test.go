package common

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v1"
	"testing"
)

type dataOptions struct {
	String  string `json:"string"`
	Integer int    `json:"integer"`
}

type testOptions struct {
	Root string       `json:"root"`
	Data *dataOptions `json:"data"`
}

type buildTest struct {
	BuildOptions `json:"options"`
}

const exampleOptionsJSON = `{
	"options": {
		"root": "value",
		"data": {
			"string": "value",
			"integer": 1
		}
	}
}`

const exampleOptionsNoDataJSON = `{
	"options": {
		"root": "value"
	}
}`

const exampleOptionsYAML = `
image: test:latest

variables:
    KEY: value
`

func (o *buildTest) Unmarshal(data string) error {
	return json.Unmarshal([]byte(data), o)
}

func TestBuildOptionsUnmarshaling(t *testing.T) {
	var options buildTest
	require.NoError(t, options.Unmarshal(exampleOptionsJSON))
	assert.Equal(t, "value", options.BuildOptions["root"])

	result, _ := options.Get("data", "string")
	assert.Equal(t, "value", result)
	result, _ = options.Get("data", "integer")
	assert.Equal(t, 1, result)

	result2, _ := options.GetString("data", "string")
	assert.Equal(t, "value", result2)
	result2, _ = options.GetString("data", "integer")
	assert.Equal(t, "", result2)
}

func TestBuildOptionsDecodeTest(t *testing.T) {
	var options buildTest
	var test testOptions
	require.NoError(t, options.Unmarshal(exampleOptionsJSON))
	require.NoError(t, options.Decode(&test))
	assert.Equal(t, "value", test.Root)
	assert.NotNil(t, test.Data)
}

func TestBuildOptionsDecodeTestNoData(t *testing.T) {
	var options buildTest
	var test testOptions
	require.NoError(t, options.Unmarshal(exampleOptionsNoDataJSON))
	require.NoError(t, options.Decode(&test))
	assert.Equal(t, "value", test.Root)
	assert.Nil(t, test.Data)
}

func TestBuildOptionsDecodeData(t *testing.T) {
	var options buildTest
	var data dataOptions
	require.NoError(t, options.Unmarshal(exampleOptionsJSON))
	require.NoError(t, options.Decode(&data, "data"))
	assert.Equal(t, "value", data.String)
	assert.Equal(t, 1, data.Integer)
}

func TestBuildOptionsSanitizeWithYamlDecode(t *testing.T) {
	options := make(BuildOptions)

	require.NoError(t, yaml.Unmarshal([]byte(exampleOptionsYAML), options))
	assert.Equal(t, BuildOptions{
		"image": "test:latest",
		"variables": map[interface{}]interface{}{
			"KEY": "value",
		},
	}, options)

	require.NoError(t, options.Sanitize())
	assert.Equal(t, BuildOptions{
		"image": "test:latest",
		"variables": map[string]interface{}{
			"KEY": "value",
		},
	}, options)
}
