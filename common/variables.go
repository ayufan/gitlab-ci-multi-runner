package common

import (
	"fmt"
	"strings"
	"errors"
	"os"
)

type BuildVariable struct {
	Key    string `json:"key"`
	Value  string `json:"value"`
	Public bool   `json:"public"`
}

type BuildVariables []BuildVariable

func (b BuildVariable) String() string {
	return fmt.Sprintf("%s=%s", b.Key, b.Value)
}

func (b BuildVariables) Public() (variables BuildVariables) {
	for _, variable := range b {
		if variable.Public {
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

func ParseVariable(text string) (variable BuildVariable, err error) {
	keyValue := strings.SplitN(text, "=", 2)
	if len(keyValue) != 2 {
		err = errors.New("missing =")
		return
	}
	variable = BuildVariable{
		Key: keyValue[0],
		Value: keyValue[1],
	}
	return
}
