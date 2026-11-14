package configreader

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// ReadJsonConfig reads a JSON file into T. Relative and absolute paths work.
// String values may contain ${{NAME}} or ${{NAME:-default}} placeholders.
func ReadJsonConfig[T any](path string) (T, error) {
	var result T
	data, err := os.ReadFile(path)
	if err != nil {
		return result, fmt.Errorf("read config %q: %w", path, err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return result, fmt.Errorf("parse JSON config %q: %w", path, err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			err = fmt.Errorf("multiple JSON values")
		}
		return result, fmt.Errorf("parse JSON config %q: %w", path, err)
	}
	missing := make(map[string]struct{})
	value = replaceJSONValues(value, missing)
	if err := missingError(missing); err != nil {
		return result, err
	}
	resolved, err := json.Marshal(value)
	if err != nil {
		return result, fmt.Errorf("encode JSON config %q: %w", path, err)
	}
	if err := json.Unmarshal(resolved, &result); err != nil {
		return result, fmt.Errorf("decode JSON config %q: %w", path, err)
	}
	return result, nil
}

func replaceJSONValues(value any, missing map[string]struct{}) any {
	switch value := value.(type) {
	case string:
		return substitute(value, missing)
	case []any:
		for i := range value {
			value[i] = replaceJSONValues(value[i], missing)
		}
	case map[string]any:
		for key, item := range value {
			value[key] = replaceJSONValues(item, missing)
		}
	}
	return value
}
