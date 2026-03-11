package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/knadh/koanf/v2"
	"gopkg.in/yaml.v3"
)

const (
	defaultCaptureLimitBytes = int64(1024 * 1024)
	defaultTimeout           = "60s"
)

// Config is the resolved runtime configuration.
type Config struct {
	LogLevel  string          `json:"log_level"`
	Output    OutputConfig    `json:"output"`
	Execution ExecutionConfig `json:"execution"`
}

// OutputConfig controls command output handling.
type OutputConfig struct {
	CaptureLimitBytes int64 `json:"capture_limit_bytes"`
}

// ExecutionConfig controls runtime behavior and safety checks.
type ExecutionConfig struct {
	DefaultTimeout time.Duration `json:"default_timeout"`
	RedactKeys     []string      `json:"redact_keys"`
	AllowPaths     []string      `json:"allow_paths"`
	DenyPaths      []string      `json:"deny_paths"`
}

// LoadOptions defines config loading behavior.
type LoadOptions struct {
	CWD           string
	ConfigPath    string
	FlagOverrides map[string]any
	Env           map[string]string
	HomeDir       string
}

// Sources tracks where the resolved config came from.
type Sources struct {
	Precedence     []string `json:"precedence"`
	UserConfig     string   `json:"user_config,omitempty"`
	ProjectConfig  string   `json:"project_config,omitempty"`
	ExplicitConfig string   `json:"explicit_config,omitempty"`
	EnvOverrides   []string `json:"env_overrides"`
	FlagOverrides  []string `json:"flag_overrides"`
}

// LoadedConfig is the full config resolution result.
type LoadedConfig struct {
	Config  Config         `json:"config"`
	Sources Sources        `json:"sources"`
	Raw     map[string]any `json:"raw"`
}

// ProjectConfigPath returns the project config location.
func ProjectConfigPath(cwd string) string {
	return filepath.Join(cwd, ".toolbox", "config.yaml")
}

// ProjectTaskDir returns the project task directory.
func ProjectTaskDir(cwd string) string {
	return filepath.Join(cwd, ".toolbox", "tasks")
}

// ProjectBundledTaskDir returns the portable project task directory.
func ProjectBundledTaskDir(cwd string) string {
	return filepath.Join(cwd, "tasks")
}

// ProjectScriptDir returns the project script directory.
func ProjectScriptDir(cwd string) string {
	return filepath.Join(cwd, ".toolbox", "scripts")
}

// ProjectBundledScriptDir returns the portable project script directory.
func ProjectBundledScriptDir(cwd string) string {
	return filepath.Join(cwd, "scripts")
}

// UserConfigPath returns the user config location.
func UserConfigPath(home string) string {
	return filepath.Join(home, ".config", "toolbox", "config.yaml")
}

// UserTaskDir returns the user task directory.
func UserTaskDir(home string) string {
	return filepath.Join(home, ".config", "toolbox", "tasks")
}

// UserScriptDir returns the user script directory.
func UserScriptDir(home string) string {
	return filepath.Join(home, ".config", "toolbox", "scripts")
}

// Load resolves configuration according to precedence.
func Load(opts LoadOptions) (LoadedConfig, error) {
	cwd := opts.CWD
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			return LoadedConfig{}, fmt.Errorf("resolve cwd: %w", err)
		}
	}
	home := opts.HomeDir
	if home == "" {
		home = os.Getenv("HOME")
		if home == "" {
			var err error
			home, err = os.UserHomeDir()
			if err != nil {
				return LoadedConfig{}, fmt.Errorf("resolve home dir: %w", err)
			}
		}
	}

	env := opts.Env
	if env == nil {
		env = envToMap(os.Environ())
	}

	k := koanf.New(".")
	mergeFlatMap(k, defaults())

	sources := Sources{
		Precedence: []string{
			"flags",
			"environment (TOOLBOX_*)",
			"project config (.toolbox/config.yaml)",
			"user config (~/.config/toolbox/config.yaml)",
			"built-in defaults",
		},
		EnvOverrides:  []string{},
		FlagOverrides: []string{},
	}

	userConfigPath := UserConfigPath(home)
	if fileExists(userConfigPath) {
		userMap, err := readYAMLMap(userConfigPath)
		if err != nil {
			return LoadedConfig{}, fmt.Errorf("load user config %s: %w", userConfigPath, err)
		}
		mergeNestedMap(k, userMap)
		sources.UserConfig = userConfigPath
	}

	projectConfigPath := ProjectConfigPath(cwd)
	if fileExists(projectConfigPath) {
		projectMap, err := readYAMLMap(projectConfigPath)
		if err != nil {
			return LoadedConfig{}, fmt.Errorf("load project config %s: %w", projectConfigPath, err)
		}
		mergeNestedMap(k, projectMap)
		sources.ProjectConfig = projectConfigPath
	}

	if opts.ConfigPath != "" {
		resolved, err := filepath.Abs(opts.ConfigPath)
		if err != nil {
			return LoadedConfig{}, fmt.Errorf("resolve --config path: %w", err)
		}
		if !fileExists(resolved) {
			return LoadedConfig{}, fmt.Errorf("config file %s does not exist", resolved)
		}
		explicitMap, err := readYAMLMap(resolved)
		if err != nil {
			return LoadedConfig{}, fmt.Errorf("load explicit config %s: %w", resolved, err)
		}
		mergeNestedMap(k, explicitMap)
		sources.ExplicitConfig = resolved
	}

	applyEnvOverrides(k, env, &sources)
	applyFlagOverrides(k, opts.FlagOverrides, &sources)

	raw := rawConfig{
		LogLevel: k.String("log_level"),
		Output: rawOutput{
			CaptureLimitBytes: k.Int64("output.capture_limit_bytes"),
		},
		Execution: rawExecution{
			DefaultTimeout: k.String("execution.default_timeout"),
			RedactKeys:     normalizeSlice(k.Strings("execution.redact_keys")),
			AllowPaths:     normalizeSlice(k.Strings("execution.allow_paths")),
			DenyPaths:      normalizeSlice(k.Strings("execution.deny_paths")),
		},
	}
	cfg, err := raw.toConfig()
	if err != nil {
		return LoadedConfig{}, err
	}

	return LoadedConfig{
		Config:  cfg,
		Sources: sources,
		Raw:     k.All(),
	}, nil
}

type rawConfig struct {
	LogLevel  string
	Output    rawOutput
	Execution rawExecution
}

type rawOutput struct {
	CaptureLimitBytes int64
}

type rawExecution struct {
	DefaultTimeout string
	RedactKeys     []string
	AllowPaths     []string
	DenyPaths      []string
}

func (r rawConfig) toConfig() (Config, error) {
	timeout := defaultTimeout
	if strings.TrimSpace(r.Execution.DefaultTimeout) != "" {
		timeout = r.Execution.DefaultTimeout
	}
	duration, err := time.ParseDuration(timeout)
	if err != nil {
		return Config{}, fmt.Errorf("invalid execution.default_timeout %q: %w", timeout, err)
	}

	captureLimit := r.Output.CaptureLimitBytes
	if captureLimit <= 0 {
		captureLimit = defaultCaptureLimitBytes
	}
	logLevel := strings.TrimSpace(r.LogLevel)
	if logLevel == "" {
		logLevel = "info"
	}

	redactKeys := normalizeSlice(r.Execution.RedactKeys)
	if len(redactKeys) == 0 {
		redactKeys = []string{"TOKEN", "SECRET", "PASSWORD", "KEY"}
	}

	return Config{
		LogLevel: logLevel,
		Output: OutputConfig{
			CaptureLimitBytes: captureLimit,
		},
		Execution: ExecutionConfig{
			DefaultTimeout: duration,
			RedactKeys:     redactKeys,
			AllowPaths:     normalizeSlice(r.Execution.AllowPaths),
			DenyPaths:      normalizeSlice(r.Execution.DenyPaths),
		},
	}, nil
}

func defaults() map[string]any {
	return map[string]any{
		"log_level":                  "info",
		"output.capture_limit_bytes": defaultCaptureLimitBytes,
		"execution.default_timeout":  defaultTimeout,
		"execution.redact_keys":      []string{"TOKEN", "SECRET", "PASSWORD", "KEY"},
		"execution.allow_paths":      []string{},
		"execution.deny_paths":       []string{},
	}
}

func applyEnvOverrides(k *koanf.Koanf, env map[string]string, sources *Sources) {
	if len(env) == 0 {
		return
	}
	keys := make([]string, 0, len(env))
	for key := range env {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if !strings.HasPrefix(key, "TOOLBOX_") {
			continue
		}
		mapped, ok := mapEnvKey(key)
		if !ok {
			continue
		}
		value := env[key]
		if mapped.isSlice {
			k.Set(mapped.key, splitCSV(value))
		} else if mapped.isInt {
			parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
			if err != nil {
				continue
			}
			k.Set(mapped.key, parsed)
		} else {
			k.Set(mapped.key, value)
		}
		sources.EnvOverrides = append(sources.EnvOverrides, key)
	}
}

func applyFlagOverrides(k *koanf.Koanf, overrides map[string]any, sources *Sources) {
	if len(overrides) == 0 {
		return
	}
	keys := make([]string, 0, len(overrides))
	for key := range overrides {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		k.Set(key, overrides[key])
		sources.FlagOverrides = append(sources.FlagOverrides, key)
	}
}

type mappedEnvKey struct {
	key     string
	isSlice bool
	isInt   bool
}

func mapEnvKey(envKey string) (mappedEnvKey, bool) {
	stripped := strings.TrimPrefix(envKey, "TOOLBOX_")
	normalized := strings.ToLower(stripped)
	switch normalized {
	case "log_level":
		return mappedEnvKey{key: "log_level"}, true
	case "output_capture_limit_bytes":
		return mappedEnvKey{key: "output.capture_limit_bytes", isInt: true}, true
	case "execution_default_timeout":
		return mappedEnvKey{key: "execution.default_timeout"}, true
	case "execution_redact_keys":
		return mappedEnvKey{key: "execution.redact_keys", isSlice: true}, true
	case "execution_allow_paths":
		return mappedEnvKey{key: "execution.allow_paths", isSlice: true}, true
	case "execution_deny_paths":
		return mappedEnvKey{key: "execution.deny_paths", isSlice: true}, true
	default:
		if !strings.Contains(normalized, "__") {
			return mappedEnvKey{}, false
		}
		return mappedEnvKey{key: strings.ReplaceAll(normalized, "__", ".")}, true
	}
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func normalizeSlice(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func envToMap(entries []string) map[string]string {
	result := make(map[string]string, len(entries))
	for _, entry := range entries {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 {
			continue
		}
		result[parts[0]] = parts[1]
	}
	return result
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func readYAMLMap(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return map[string]any{}, nil
	}
	var parsed map[string]any
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		return nil, err
	}
	if parsed == nil {
		parsed = map[string]any{}
	}
	return parsed, nil
}

func mergeFlatMap(k *koanf.Koanf, values map[string]any) {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		k.Set(key, values[key])
	}
}

func mergeNestedMap(k *koanf.Koanf, values map[string]any) {
	flattenNestedMap("", values, func(key string, value any) {
		k.Set(key, value)
	})
}

func flattenNestedMap(prefix string, value any, setFn func(string, any)) {
	switch typed := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			next := key
			if prefix != "" {
				next = prefix + "." + key
			}
			flattenNestedMap(next, typed[key], setFn)
		}
	case map[any]any:
		converted := make(map[string]any, len(typed))
		for key, item := range typed {
			converted[fmt.Sprint(key)] = item
		}
		flattenNestedMap(prefix, converted, setFn)
	default:
		if prefix != "" {
			setFn(prefix, value)
		}
	}
}
