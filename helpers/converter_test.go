package helpers

import (
	"gopkg.in/yaml.v1"
	"reflect"
	"testing"
)

type TestObj struct {
	Text   string `json:"TextJson" yaml:"TextYaml"`
	Number int
}

func TestSimpleJsonMarshalling(t *testing.T) {

	jsonString := ToJson(TestObj{
		Text:   "example",
		Number: 25,
	})
	expectedJson := "{\"TextJson\":\"example\",\"Number\":25}"

	if jsonString != expectedJson {
		t.Error("Expected ", expectedJson, ", got ", jsonString)
	}
}

func TestSimpleYamlMarshalling(t *testing.T) {

	ymlString := ToYAML(TestObj{
		Text:   "example",
		Number: 25,
	})
	expectedYml := "TextYaml: example\nnumber: 25\n"

	if ymlString != expectedYml {
		t.Error("Expected ", expectedYml, ", got ", ymlString)
	}
}

func TestSimpleTomlMarshalling(t *testing.T) {

	tomlString := ToTOML(TestObj{
		Text:   "example",
		Number: 25,
	})
	expectedToml := "Text = \"example\"\nNumber = 25\n"

	if tomlString != expectedToml {
		t.Error("Expected ", expectedToml, ", got ", tomlString)
	}
}

func TestToConfigMap(t *testing.T) {
	data := `
build:
    script:
         - echo "1" >> foo
         - cat foo

cache:
    untracked: true
    paths:
        - vendor/
        - foo

test:
    script:
    - make test
`

	config := make(map[string]interface{})
	err := yaml.Unmarshal([]byte(data), config)
	if err != nil {
		t.Error("Error parsing test YAML data")
	}

	expectedCacheConfig := map[string]interface{}{
		"untracked": true,
		"paths":     []interface{}{"vendor/", "foo"},
	}
	cacheConfig, ok := ToConfigMap(config["cache"])

	if !ok {
		t.Error("Conversion failed")
	}

	if !reflect.DeepEqual(cacheConfig, expectedCacheConfig) {
		t.Error("Result ", cacheConfig, " was not equal to ", expectedCacheConfig)
	}
}
