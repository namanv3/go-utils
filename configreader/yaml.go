package configreader

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ReadYamlConfig reads a YAML file into T. Relative and absolute paths work.
// String values may contain ${{NAME}} or ${{NAME:-default}} placeholders.
func ReadYamlConfig[T any](path string) (T, error) {
	var result T
	data, err := os.ReadFile(path)
	if err != nil {
		return result, fmt.Errorf("read config %q: %w", path, err)
	}
	var document yaml.Node
	if err := yaml.Unmarshal(data, &document); err != nil {
		return result, fmt.Errorf("parse YAML config %q: %w", path, err)
	}
	missing := make(map[string]struct{})
	replaceYAMLValues(&document, missing)
	if err := missingError(missing); err != nil {
		return result, err
	}
	if err := document.Decode(&result); err != nil {
		return result, fmt.Errorf("decode YAML config %q: %w", path, err)
	}
	return result, nil
}

func replaceYAMLValues(node *yaml.Node, missing map[string]struct{}) {
	switch node.Kind {
	case yaml.DocumentNode, yaml.SequenceNode:
		for _, child := range node.Content {
			replaceYAMLValues(child, missing)
		}
	case yaml.MappingNode:
		for i := 1; i < len(node.Content); i += 2 {
			replaceYAMLValues(node.Content[i], missing)
		}
	case yaml.ScalarNode:
		if node.Tag != "!!str" {
			return
		}
		original := node.Value
		node.Value = substitute(original, missing)
		// A plain scalar made entirely of a placeholder can represent a number
		// or boolean; quoted scalars and mixed text remain strings.
		if node.Style == 0 && placeholder.FindString(original) == original {
			node.Tag = ""
		}
	}
}
