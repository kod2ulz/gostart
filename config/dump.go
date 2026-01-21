package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/iancoleman/strcase"
	"gopkg.in/yaml.v3"
)

// ConfigFormat represents the output format for configuration
type ConfigFormat string

const (
	FormatJSON ConfigFormat = "json"
	FormatYAML ConfigFormat = "yaml"
	FormatENV  ConfigFormat = "env"
)

// ConfigEntry represents a single configuration entry with its source
type ConfigEntry struct {
	Key            string     `json:"key" yaml:"key"`
	Value          string     `json:"value" yaml:"value"`
	Source         string     `json:"source" yaml:"source"`
	Type           string     `json:"type,omitempty" yaml:"type,omitempty"`
	OriginalKey    string     `json:"original_key,omitempty" yaml:"original_key,omitempty"` // Original env var name or config key
	ExpiresAt      *time.Time `json:"expires_at,omitempty" yaml:"expires_at,omitempty"`
	TimeToLive     time.Duration `json:"time_to_live,omitempty" yaml:"time_to_live,omitempty"`
	IsStale        bool       `json:"is_stale" yaml:"is_stale,omitempty"`
}

// ConfigDump represents the complete configuration dump
type ConfigDump struct {
	Entries []ConfigEntry `json:"entries" yaml:"entries"`
}

// DumpConfig outputs the current configuration in the specified format
func DumpConfig(format ConfigFormat) error {
	dump, err := CollectConfig()
	if err != nil {
		return fmt.Errorf("failed to collect config: %w", err)
	}

	switch format {
	case FormatJSON:
		return dump.PrintJSON()
	case FormatYAML:
		return dump.PrintYAML()
	case FormatENV:
		return dump.PrintENV()
	default:
		return fmt.Errorf("unknown format: %s (supported: json, yaml, env)", format)
	}
}

// DumpCachedConfig outputs only the cached configuration in the specified format
func DumpCachedConfig(format ConfigFormat) error {
	dump, err := CollectCachedConfig()
	if err != nil {
		return fmt.Errorf("failed to collect cached config: %w", err)
	}

	switch format {
	case FormatJSON:
		return dump.PrintJSON()
	case FormatYAML:
		return dump.PrintYAML()
	case FormatENV:
		return dump.PrintENV()
	default:
		return fmt.Errorf("unknown format: %s (supported: json, yaml, env)", format)
	}
}

// CollectConfig gathers all configuration values from all sources
func CollectConfig() (*ConfigDump, error) {
	dump := &ConfigDump{}

	// Collect from all sources in priority order
	// Note: This will show what WOULD be loaded, not necessarily what's currently being used

	// 1. YAML config
	if yamlEntries := collectYAMLConfig(); len(yamlEntries) > 0 {
		dump.Entries = append(dump.Entries, yamlEntries...)
	}

	// 2. Environment variables
	if envEntries := collectEnvConfig(); len(envEntries) > 0 {
		dump.Entries = append(dump.Entries, envEntries...)
	}

	// 3. You could add DB and Vault collectors here if needed

	return dump, nil
}

// CollectCachedConfig gathers only the cached configuration values (DB, Vault)
// with their expiry times and cache metadata
func CollectCachedConfig() (*ConfigDump, error) {
	dump := &ConfigDump{}

	// Collect cached Vault values
	vaultEntries := Vault.InspectCache()
	dump.Entries = append(dump.Entries, vaultEntries...)

	// Collect cached DB values
	dbEntries := DB.InspectCache()
	dump.Entries = append(dump.Entries, dbEntries...)

	return dump, nil
}

// CollectEffectiveConfig shows the full picture - what's actually loaded from each source
// including cached values, YAML config, and environment variables
func CollectEffectiveConfig() (*ConfigDump, error) {
	dump := &ConfigDump{}

	// 1. YAML config (static, not cached)
	if yamlEntries := collectYAMLConfig(); len(yamlEntries) > 0 {
		dump.Entries = append(dump.Entries, yamlEntries...)
	}

	// 2. Environment variables (static, not cached)
	if envEntries := collectEnvConfig(); len(envEntries) > 0 {
		dump.Entries = append(dump.Entries, envEntries...)
	}

	// 3. Cached DB values (with expiry info)
	if dbEntries := DB.InspectCache(); len(dbEntries) > 0 {
		dump.Entries = append(dump.Entries, dbEntries...)
	}

	// 4. Cached Vault values (with expiry info)
	if vaultEntries := Vault.InspectCache(); len(vaultEntries) > 0 {
		dump.Entries = append(dump.Entries, vaultEntries...)
	}

	return dump, nil
}

// collectYAMLConfig collects configuration from YAML file
func collectYAMLConfig() []ConfigEntry {
	entries := []ConfigEntry{}

	// Flatten the yamlData map
	if yamlData != nil {
		for key, value := range flattenMap(yamlData, "") {
			entries = append(entries, ConfigEntry{
				Key:    key,
				Value:  formatValue(value),
				Source: "yaml",
				Type:   inferTypeFromValue(value),
			})
		}
	}

	return entries
}

// flattenMap recursively flattens a nested map into dot-separated keys
func flattenMap(data map[string]interface{}, prefix string) map[string]interface{} {
	result := make(map[string]interface{})

	for key, value := range data {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		switch v := value.(type) {
		case map[string]interface{}:
			// Recursively flatten nested maps
			nested := flattenMap(v, fullKey)
			for nk, nv := range nested {
				result[nk] = nv
			}
		default:
			result[fullKey] = value
		}
	}

	return result
}

// formatValue converts a value to its string representation
func formatValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case bool:
		return fmt.Sprintf("%t", v)
	case int, int32, int64:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return fmt.Sprintf("%f", v)
	case []interface{}:
		// Format as array
		parts := make([]string, len(v))
		for i, item := range v {
			parts[i] = formatValue(item)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	default:
		return fmt.Sprintf("%v", v)
	}
}

// inferTypeFromValue infers the type from an interface{} value
func inferTypeFromValue(value interface{}) string {
	switch value.(type) {
	case bool:
		return "boolean"
	case int, int32, int64:
		return "integer"
	case float32, float64:
		return "float"
	case string:
		return "string"
	case []interface{}:
		return "array"
	case map[string]interface{}:
		return "object"
	default:
		return "unknown"
	}
}

// collectEnvConfig collects all environment variables and converts them to config format
// This shows what environment variables would be available to the config system
func collectEnvConfig() []ConfigEntry {
	return collectEnvConfigWithPrefix()
}

// collectEnvConfigWithPrefix collects environment variables with optional prefix filtering
// If prefixes are provided, only vars matching those prefixes are collected
func collectEnvConfigWithPrefix(prefixes ...string) []ConfigEntry {
	// Get all environment variables
	envMap := make(map[string]string)
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	entries := []ConfigEntry{}

	// Collect environment variables
	for key, value := range envMap {
		// Skip system environment variables that are clearly not config
		if isSystemEnv(key) {
			continue
		}

		// If prefixes specified, only collect vars that start with one of them
		if len(prefixes) > 0 {
			matches := false
			for _, prefix := range prefixes {
				if strings.HasPrefix(key, prefix) {
					matches = true
					break
				}
			}
			if !matches {
				continue
			}
		}

		// Convert SCREAMING_SNAKE_CASE to camelCase.dot.notation
		configKey := toConfigKey(key)

		entries = append(entries, ConfigEntry{
			Key:         configKey,
			Value:       value,
			Source:      "environment",
			Type:        inferType(value),
			OriginalKey: key, // Store the original env var name for reference
		})
	}

	return entries
}

// isSystemEnv filters out system environment variables that aren't config
func isSystemEnv(key string) bool {
	systemPrefixes := []string{
		// System paths
		"PATH", "HOME", "USER", "SHELL", "TERM", "PWD", "TMPDIR", "TMP",
		"LANG", "LC_", "LOGNAME", "TZ", "BLOCKSIZE",

		// Shell/terminal
		"EDITOR", "PAGER", "VISUAL", "LS_", "HIST", "MAIL", "MANPAGER",

		// SSH/Git
		"SSH_", "GIT_",

		// Go
		"GOPATH", "GOROOT", "GOOS", "GOARCH", "GOMODCACHE", "GOPROXY", "GO111MODULE",
		"CGO_", "GO",

		// macOS/iOS
		"APPLE_", "COMMAND_MODE_", "SECURITYSESSIONID",

		// Linux/X11
		"XDG_", "DISPLAY", "DBUS_", "VTE_", "WINDOWID", "WAYLAND_",

		// Docker/Podman
		"DOCKER_", "KUBERNETES_", "KUBE_",

		// CI/CD
		"CI_", "GITHUB_", "GITLAB_", "JENKINS_",
		"BUILD_", "RUNNER_", "WORKFLOW_",

		// Cloud providers
		"AWS_", "AZURE_", "GOOGLE_", "GCP_",
		"VCAP_", "CF_", "HEROKU_",

		// System internals
		"SHLVL", "_", "__CF_", "SECURITYSESSIONID",
	}

	for _, prefix := range systemPrefixes {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}

	return false
}

// toConfigKey converts ENV_VAR_NAME to camelCase.dot.notation
func toConfigKey(envKey string) string {
	parts := strings.Split(envKey, "_")

	// First part is usually the prefix (APP, HTTP_SERVER, etc.)
	// Convert to lowercase and keep as prefix
	if len(parts) > 0 {
		parts[0] = strings.ToLower(parts[0])
	}

	// Convert rest from SCREAMING_SNAKE to camelCase
	for i := 1; i < len(parts); i++ {
		parts[i] = strcase.ToLowerCamel(parts[i])
	}

	// Join with dots
	return strings.Join(parts, ".")
}

// inferType attempts to infer the type from a string value
func inferType(value string) string {
	// Try to parse as different types to guess the type
	if value == "true" || value == "false" {
		return "boolean"
	}

	if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
		return "array"
	}

	// Could add more sophisticated type detection here
	return "string"
}

// PrintJSON outputs the configuration as JSON
func (d *ConfigDump) PrintJSON() error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(d)
}

// PrintYAML outputs the configuration as nested YAML (like a config file)
func (d *ConfigDump) PrintYAML() error {
	// Convert flat entries to nested map structure
	nested := d.toNestedMap()
	return yaml.NewEncoder(os.Stdout).Encode(nested)
}

// toNestedMap converts flat entries to nested map structure
func (d *ConfigDump) toNestedMap() map[string]interface{} {
	result := make(map[string]interface{})

	for _, entry := range d.Entries {
		// Skip empty keys
		if entry.Key == "" {
			continue
		}

		// Split by dots to get nested structure
		parts := strings.Split(entry.Key, ".")
		current := result

		// Navigate/create nested structure
		for i, part := range parts {
			if i == len(parts)-1 {
				// Last part - set the value
				current[part] = convertValue(entry.Value, entry.Type)
			} else {
				// Middle part - navigate or create map
				if _, exists := current[part]; !exists {
					current[part] = make(map[string]interface{})
				}
				if next, ok := current[part].(map[string]interface{}); ok {
					current = next
				} else {
					// If it's not a map, we have a conflict - replace with map
					current[part] = make(map[string]interface{})
					current = current[part].(map[string]interface{})
				}
			}
		}
	}

	return result
}

// convertValue converts string value to appropriate type
func convertValue(value string, typeHint string) interface{} {
	switch typeHint {
	case "boolean":
		return value == "true"
	case "integer":
		var i int64
		fmt.Sscanf(value, "%d", &i)
		return i
	case "float":
		var f float64
		fmt.Sscanf(value, "%f", &f)
		return f
	case "array":
		// Try to parse as JSON array
		var result []interface{}
		json.Unmarshal([]byte(value), &result)
		return result
	default:
		return value
	}
}

// PrintENV outputs the configuration in .env file format
// Shows both the config key (as comment) and the environment variable format
func (d *ConfigDump) PrintENV() error {
	for _, entry := range d.Entries {
		// Show config key as comment for reference
		if entry.Source == "environment" && entry.OriginalKey != "" {
			// For env vars, show: # Config key: staff.admin.seeder.run
			fmt.Printf("# Config key: %s\n", entry.Key)
			// Show the original env var name and value
			fmt.Printf("%s=%s\n", entry.OriginalKey, entry.Value)
		} else {
			// For other sources, show what env var would set it
			envKey := toEnvKey(entry.Key)
			fmt.Printf("# %s (from %s)\n", entry.Key, entry.Source)
			fmt.Printf("%s=%s\n", envKey, entry.Value)
		}
		fmt.Println() // Blank line for readability
	}
	return nil
}

// toEnvKey converts config.key notation to ENV_VAR notation
func toEnvKey(configKey string) string {
	parts := strings.Split(configKey, ".")

	// First part to uppercase
	if len(parts) > 0 {
		parts[0] = strings.ToUpper(parts[0])
	}

	// Convert rest to SCREAMING_SNAKE_CASE
	for i := 1; i < len(parts); i++ {
		parts[i] = strcase.ToScreamingSnake(parts[i])
	}

	return strings.Join(parts, "_")
}
