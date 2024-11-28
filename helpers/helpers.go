package helpers

import (
	"encoding/json"
)

func Ptr[T any](t T) *T {
	return &t
}

func FromByteArray[T any](tokenBytes []byte) (T, error) {
	var obj T
	err := json.Unmarshal(tokenBytes, &obj)
	return obj, err
}

func Contains[T comparable](list []T, elementToFind T) bool {
	for _, elem := range list {
		if elem == elementToFind {
			return true
		}
	}
	return false
}
