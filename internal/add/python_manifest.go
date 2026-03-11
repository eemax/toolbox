package add

import (
	"fmt"

	"gopkg.in/yaml.v3"

	"toolbox/internal/manifest"
)

func buildTaskManifest(cfg resolvedPythonConfig) (manifest.Task, []byte, error) {
	args := make([]string, 0, len(cfg.Args)+1)
	args = append(args, cfg.ScriptDestPath)
	args = append(args, cfg.Args...)

	task := manifest.Task{
		Name:        cfg.Name,
		Description: cfg.Description,
		Command:     cfg.PythonBin,
		Args:        args,
		Input:       manifest.InputSpec{Mode: cfg.InputMode},
		Output:      manifest.OutputSpec{Mode: cfg.OutputMode},
		TimeoutRaw:  cfg.Timeout,
		Env:         cloneMap(cfg.Env),
		Requires:    []string{cfg.PythonBin},
		Tags:        append([]string(nil), cfg.Tags...),
		CWD:         cfg.TaskCWD,
	}

	type taskFile struct {
		Name        string              `yaml:"name"`
		Description string              `yaml:"description,omitempty"`
		Command     string              `yaml:"command"`
		Args        []string            `yaml:"args"`
		Input       manifest.InputSpec  `yaml:"input"`
		Output      manifest.OutputSpec `yaml:"output"`
		Timeout     string              `yaml:"timeout,omitempty"`
		Env         map[string]string   `yaml:"env,omitempty"`
		Requires    []string            `yaml:"requires,omitempty"`
		Tags        []string            `yaml:"tags,omitempty"`
		CWD         string              `yaml:"cwd,omitempty"`
	}

	fileModel := taskFile{
		Name:        task.Name,
		Description: task.Description,
		Command:     task.Command,
		Args:        task.Args,
		Input:       task.Input,
		Output:      task.Output,
		Timeout:     task.TimeoutRaw,
		Env:         cloneMap(task.Env),
		Requires:    append([]string(nil), task.Requires...),
		Tags:        append([]string(nil), task.Tags...),
		CWD:         task.CWD,
	}

	data, err := yaml.Marshal(fileModel)
	if err != nil {
		return manifest.Task{}, nil, fmt.Errorf("marshal task manifest: %w", err)
	}
	if len(data) == 0 || data[len(data)-1] != '\n' {
		data = append(data, '\n')
	}
	return task, data, nil
}

func cloneMap(value map[string]string) map[string]string {
	if len(value) == 0 {
		return nil
	}
	out := make(map[string]string, len(value))
	for key, val := range value {
		out[key] = val
	}
	return out
}
