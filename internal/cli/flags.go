package cli

type globalFlags struct {
	ConfigPath string
	Verbose    bool
	LogLevel   string
	JSON       bool
}

type runFlags struct {
	InputFile     string
	DryRun        bool
	DryRunFullEnv bool
	Timeout       string
}

type addPythonFlags struct {
	FromSpec    string
	Name        string
	Script      string
	Description string
	Args        []string
	Env         []string
	Timeout     string
	CWD         string
	Tags        []string
	InputMode   string
	OutputMode  string
	PythonBin   string
	Scope       string
	Overwrite   bool
}
