package add

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func loadSpec(path string) (pythonSpec, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return pythonSpec{}, fmt.Errorf("resolve --from-spec path: %w", err)
	}
	data, err := os.ReadFile(absPath)
	if err != nil {
		return pythonSpec{}, fmt.Errorf("read --from-spec file %q: %w", absPath, err)
	}

	var spec pythonSpec
	switch strings.ToLower(filepath.Ext(absPath)) {
	case ".json":
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&spec); err != nil {
			return pythonSpec{}, fmt.Errorf("parse --from-spec JSON %q: %w", absPath, err)
		}
		if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
			if err == nil {
				return pythonSpec{}, fmt.Errorf("parse --from-spec JSON %q: multiple documents are not allowed", absPath)
			}
			return pythonSpec{}, fmt.Errorf("parse --from-spec JSON %q: %w", absPath, err)
		}
	default:
		decoder := yaml.NewDecoder(bytes.NewReader(data))
		decoder.KnownFields(true)
		if err := decoder.Decode(&spec); err != nil {
			return pythonSpec{}, fmt.Errorf("parse --from-spec YAML %q: %w", absPath, err)
		}
		var extra any
		if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
			if err == nil {
				return pythonSpec{}, fmt.Errorf("parse --from-spec YAML %q: multiple documents are not allowed", absPath)
			}
			return pythonSpec{}, fmt.Errorf("parse --from-spec YAML %q: %w", absPath, err)
		}
	}

	if strings.TrimSpace(spec.APIVersion) != specAPIVersion {
		return pythonSpec{}, fmt.Errorf("--from-spec api_version must be %q", specAPIVersion)
	}
	return spec, nil
}

func valueFromSpec(spec *pythonSpec, getter func(*pythonSpec) string) string {
	if spec == nil {
		return ""
	}
	return getter(spec)
}

func sliceFromSpec(spec *pythonSpec, getter func(*pythonSpec) []string) []string {
	if spec == nil {
		return nil
	}
	return getter(spec)
}
