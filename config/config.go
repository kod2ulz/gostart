package config

import (
	"fmt"
	"strings"
)

// Get retrieves a configuration value by searching through the configured sources
// in the specified order of precedence: Vault -> DB -> YAML -> Env -> Default.
func Get(key string, defaultValue ...interface{}) Value {

	// 1. check for special cases
	switch k := strings.ToLower(key); k {
	case "host":
		return Value(Env.GetHost())
	case "version":
		return Env.GetVersion()
	}

	// 1. Check Vault
	if val := Vault.Get(key); val.Valid() {
		return val
	}

	// 2. Check Database
	if val := DB.Get(key); val.Valid() {
		return val
	}

	// 3. Check YAML
	if val := Yaml.Get(key); val.Valid() {
		return val
	}

	// 4. Check Environment Variables
	if val := Env.Helper().Get(key); val.Valid() {
		return val
	}

	// 5. Fallback to default
	if len(defaultValue) > 0 {
		return Value(fmt.Sprint(defaultValue[0]))
	}

	return ""
}
