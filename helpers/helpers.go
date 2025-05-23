package helpers

import (
	"encoding/json"
	"slices"
)

func Ptr[T any](t T) *T {
	return &t
}

func TrueorNil(b bool) *bool {
	if b {
		return &b
	}
	return nil
}

func FromByteArray[T any](tokenBytes []byte) (T, error) {
	var obj T
	err := json.Unmarshal(tokenBytes, &obj)
	return obj, err
}

func Contains[T comparable](list []T, elementToFind T) bool {
	return slices.Contains(list, elementToFind)
}

func ListToMap[T any, K comparable](list []T, keyBuilder func(t T) K) map[K]T {
	result := map[K]T{}
	for _, element := range list {
		result[keyBuilder(element)] = element
	}
	return result
}

func Filter[T any](list []T, filter func(elem T) bool) []T {
	res := []T{}
	for _, elem := range list {
		if filter(elem) {
			res = append(res, elem)
		}
	}
	return res
}

func FirstInList[T any](list []T, filter func(elem T) bool) *T {
	for _, elem := range list {
		if filter(elem) {
			return &elem
		}
	}
	return nil
}

func Map[T any, S any](list []T, mapper func(elem T) S) []S {
	res := []S{}
	for _, elem := range list {
		res = append(res, mapper(elem))
	}
	return res
}
