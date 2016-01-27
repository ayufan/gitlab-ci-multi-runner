package common

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

type BuildVariable struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	Public   bool   `json:"public"`
	Internal bool   `json:"-"`
	File     bool   `json:"-"`
}

type BuildVariables []BuildVariable

func (b BuildVariable) String() string {
	return fmt.Sprintf("%s=%s", b.Key, b.Value)
}

func (b BuildVariables) PublicOrInternal() (variables BuildVariables) {
	for _, variable := range b {
		if variable.Public || variable.Internal {
			variables = append(variables, variable)
		}
	}
	return variables
}

func (b BuildVariables) StringList() (variables []string) {
	for _, variable := range b {
		variables = append(variables, variable.String())
	}
	return variables
}

func (b BuildVariables) Get(key string) string {
	switch key {
	case "$":
		return key
	case "*", "#", "@", "!", "?", "0", "1", "2", "3", "4", "5", "6", "7", "8", "9":
		return ""
	}
	for i := len(b) - 1; i >= 0; i-- {
		if b[i].Key == key {
			return b[i].Value
		}
	}
	return ""
}

func (b BuildVariables) ExpandValue(value string) string {
	return os.Expand(value, b.Get)
}

func (b BuildVariables) Expand() (variables BuildVariables) {
	for _, variable := range b {
		variable.Value = b.ExpandValue(variable.Value)
		variables = append(variables, variable)
	}
	return variables
}

func ParseVariable(text string) (variable BuildVariable, err error) {
	keyValue := strings.SplitN(text, "=", 2)
	if len(keyValue) != 2 {
		err = errors.New("missing =")
		return
	}
	variable = BuildVariable{
		Key:   keyValue[0],
		Value: keyValue[1],
	}
	return
}
