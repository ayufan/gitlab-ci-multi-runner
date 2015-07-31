package helpers

import (
	"strings"
)

func ExtractRegistry(imageName string) *string {

	nameParts := strings.Split(imageName, "/")

	if len(nameParts) == 3 {
		return &nameParts[0]
	}

	return nil
}
