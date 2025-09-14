package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// yamlData holds the parsed YAML configuration.
var yamlData map[string]interface{}

// Yaml provides methods for accessing YAML configuration.
var Yaml yamlSource

type yamlSource struct{}

// Load reads and parses a YAML file from the given path.
// It populates the package-level yamlData map.
func (y yamlSource) Load(path string) error {
	file, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var data map[string]interface{}
	if err := yaml.Unmarshal(file, &data); err != nil {
		return err
	}

	yamlData = data
	return nil
}

// Get retrieves a value from the loaded YAML data using a dot-separated key.
// It returns a Value type that can be converted to string, int, bool, etc.
func (y yamlSource) Get(key string, defaultValue ...interface{}) Value {
	val := y.get(key)
	if val != nil {
		return Value(fmt.Sprint(val))
	}

	if len(defaultValue) > 0 {
		return Value(fmt.Sprint(defaultValue[0]))
	}
	return ""
}

// get recursively searches for a key within the nested map.
func (y yamlSource) get(key string) interface{} {
	keys := strings.Split(key, ".")
	var current interface{} = yamlData

	for _, k := range keys {
		if asMap, ok := current.(map[string]interface{}); ok {
			if val, found := asMap[k]; found {
				current = val
			} else {
				return nil // Key not found
			}
		} else {
			return nil // Not a map, cannot go deeper
		}
	}

	return current
}
