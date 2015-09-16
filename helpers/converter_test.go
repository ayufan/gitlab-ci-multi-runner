package helpers

import (
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
