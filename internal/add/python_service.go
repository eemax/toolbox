package add

import (
	"fmt"
	"os/exec"
	"strings"

	"toolbox/internal/manifest"
)

type pythonAdder struct {
	lookPath    func(string) (string, error)
	compile     func(string, string) (string, error)
	loadCatalog func(manifest.LoadOptions) manifest.Catalog
}

// AddPython creates a python task manifest and copies the script into toolbox-managed paths.
func AddPython(opts PythonOptions) (PythonResult, error) {
	adder := pythonAdder{
		lookPath:    exec.LookPath,
		compile:     compilePythonScript,
		loadCatalog: manifest.Load,
	}
	return adder.add(opts)
}

func (a pythonAdder) add(opts PythonOptions) (PythonResult, error) {
	cfg, err := a.resolveOptions(opts)
	if err != nil {
		return PythonResult{}, err
	}

	if err := a.checkTaskNameConflicts(cfg); err != nil {
		return PythonResult{}, err
	}

	manifestExists := fileExists(cfg.ManifestPath)
	scriptExists := fileExists(cfg.ScriptDestPath)
	if manifestExists && !cfg.Overwrite {
		return PythonResult{}, fmt.Errorf("task file %q already exists; rerun with --overwrite to replace it", cfg.ManifestPath)
	}
	if scriptExists && !cfg.Overwrite {
		return PythonResult{}, fmt.Errorf("script file %q already exists; rerun with --overwrite to replace it", cfg.ScriptDestPath)
	}

	interpreterPath, err := a.lookPath(cfg.PythonBin)
	if err != nil {
		return PythonResult{}, fmt.Errorf("python interpreter %q was not found in PATH", cfg.PythonBin)
	}

	compileOutput, err := a.compile(interpreterPath, cfg.SourceScript)
	if err != nil {
		trimmed := strings.TrimSpace(compileOutput)
		if trimmed == "" {
			return PythonResult{}, fmt.Errorf("py_compile check failed for %q: %w", cfg.SourceScript, err)
		}
		return PythonResult{}, fmt.Errorf("py_compile check failed for %q: %v (%s)", cfg.SourceScript, err, trimmed)
	}

	task, manifestBytes, err := buildTaskManifest(cfg)
	if err != nil {
		return PythonResult{}, err
	}
	if err := manifest.ValidateTask(task); err != nil {
		return PythonResult{}, fmt.Errorf("generated manifest is invalid: %w", err)
	}

	scriptBytes, scriptPerm, err := readScript(cfg.SourceScript)
	if err != nil {
		return PythonResult{}, err
	}

	overwritten, err := writeScaffoldFiles(cfg, scriptBytes, scriptPerm, manifestBytes)
	if err != nil {
		return PythonResult{}, err
	}

	return PythonResult{
		Status:       "created",
		Task:         cfg.Name,
		Scope:        cfg.Scope,
		ManifestPath: cfg.ManifestPath,
		ScriptPath:   cfg.ScriptDestPath,
		Overwritten:  overwritten,
		PythonBin:    cfg.PythonBin,
		Checks: []Check{
			{Name: "interpreter", Status: "ok"},
			{Name: "py_compile", Status: "ok"},
			{Name: "manifest", Status: "ok"},
		},
		NextCommand: fmt.Sprintf("toolbox run %s", cfg.Name),
	}, nil
}
