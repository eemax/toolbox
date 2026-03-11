package add

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func readScript(path string) ([]byte, os.FileMode, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, 0, fmt.Errorf("read source script %q: %w", path, err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, 0, fmt.Errorf("stat source script %q: %w", path, err)
	}
	perm := info.Mode().Perm()
	if perm == 0 {
		perm = 0o644
	}
	return data, perm, nil
}

func writeScaffoldFiles(cfg resolvedPythonConfig, script []byte, scriptPerm os.FileMode, manifestBytes []byte) (bool, error) {
	if err := os.MkdirAll(cfg.ScriptDir, 0o755); err != nil {
		return false, fmt.Errorf("create script directory %q: %w", cfg.ScriptDir, err)
	}
	if err := os.MkdirAll(cfg.TaskDir, 0o755); err != nil {
		return false, fmt.Errorf("create task directory %q: %w", cfg.TaskDir, err)
	}

	previousScript, hadScript, err := snapshotFile(cfg.ScriptDestPath)
	if err != nil {
		return false, err
	}
	_, hadManifest, err := snapshotFile(cfg.ManifestPath)
	if err != nil {
		return false, err
	}

	if err := atomicWriteFile(cfg.ScriptDestPath, script, scriptPerm); err != nil {
		return false, err
	}
	if err := atomicWriteFile(cfg.ManifestPath, manifestBytes, 0o644); err != nil {
		if hadScript {
			_ = atomicWriteFile(cfg.ScriptDestPath, previousScript.Data, previousScript.Perm)
		} else {
			_ = os.Remove(cfg.ScriptDestPath)
		}
		return false, err
	}

	return hadScript || hadManifest, nil
}

func atomicWriteFile(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".toolbox-add-*")
	if err != nil {
		return fmt.Errorf("create temporary file in %q: %w", dir, err)
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temporary file for %q: %w", path, err)
	}
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("chmod temporary file for %q: %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temporary file for %q: %w", path, err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace %q: %w", path, err)
	}
	return nil
}

func snapshotFile(path string) (existingFile, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return existingFile{}, false, nil
		}
		return existingFile{}, false, fmt.Errorf("read existing file %q: %w", path, err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return existingFile{}, false, fmt.Errorf("stat existing file %q: %w", path, err)
	}
	return existingFile{Data: data, Perm: info.Mode().Perm()}, true, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
