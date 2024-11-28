package helpers

import (
	"fmt"

	"github.com/google/uuid"
)

func GenerateUUID(prefix string) string {
	if prefix == "" {
		return uuid.NewString()
	}
	return fmt.Sprintf("%s-%s", prefix, uuid.NewString())
}
