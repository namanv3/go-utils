// Package configreader loads YAML and JSON configuration files with environment
// variable placeholders in string values.
package configreader

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

var placeholder = regexp.MustCompile(`\$\{\{([A-Za-z_][A-Za-z0-9_]*)(?::-([^}]*))?\}\}`)

func ConfigReader[T any](path string) (T, error) {
	var result T
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yaml", ".yml":
		return ReadYamlConfig[T](path)
	case ".json":
		return ReadJsonConfig[T](path)
	default:
		return result, fmt.Errorf("unsupported config file extension %q (want .yaml, .yml, or .json)", filepath.Ext(path))
	}
}

// Set, empty environment variables take precedence over defaults.
func substitute(value string, missing map[string]struct{}) string {
	return placeholder.ReplaceAllStringFunc(value, func(token string) string {
		indices := placeholder.FindStringSubmatchIndex(token)
		name := token[indices[2]:indices[3]]
		if value, ok := os.LookupEnv(name); ok {
			return value
		}
		if indices[4] >= 0 {
			return token[indices[4]:indices[5]]
		}
		missing[name] = struct{}{}
		return token
	})
}

func missingError(missing map[string]struct{}) error {
	if len(missing) == 0 {
		return nil
	}
	names := make([]string, 0, len(missing))
	for name := range missing {
		names = append(names, name)
	}
	slices.Sort(names)
	return fmt.Errorf("missing environment variables: %s", strings.Join(names, ", "))
}
