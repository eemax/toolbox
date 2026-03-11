package toolbox

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
)

func TestScripts(t *testing.T) {
	projectRoot, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	testscript.Run(t, testscript.Params{
		Dir: filepath.Join(projectRoot, "testdata", "scripts"),
		Setup: func(env *testscript.Env) error {
			bin := filepath.Join(env.WorkDir, "toolbox")
			build := exec.Command("go", "build", "-o", bin, "./cmd/toolbox")
			build.Dir = projectRoot
			build.Env = os.Environ()
			if output, err := build.CombinedOutput(); err != nil {
				return &buildError{err: err, output: string(output)}
			}
			home := filepath.Join(env.WorkDir, "home")
			if err := os.MkdirAll(filepath.Join(home, ".config", "toolbox", "tasks"), 0o755); err != nil {
				return err
			}
			env.Setenv("TOOLBOX_BIN", bin)
			env.Setenv("HOME", home)
			env.Setenv("PROJECT_ROOT", projectRoot)
			return nil
		},
	})
}

type buildError struct {
	err    error
	output string
}

func (e *buildError) Error() string {
	return e.err.Error() + "\n" + e.output
}
