package pnpminstall

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/paketo-buildpacks/packit/v2/fs"
	"github.com/paketo-buildpacks/packit/v2/pexec"
	"github.com/paketo-buildpacks/packit/v2/scribe"
)

func NewInstallBuildProcess(executable Executable, environment EnvironmentConfig, logger scribe.Logger) InstallBuildProcess {
	return InstallBuildProcess{
		executable:  executable,
		environment: environment,
		logger:      logger,
	}
}

type InstallBuildProcess struct {
	executable  Executable
	environment EnvironmentConfig
	logger      scribe.Logger
}

func (r InstallBuildProcess) ShouldRun(workingDir string, metadata map[string]interface{}, npmrcPath string) (bool, string, error) {
	return true, "", nil
}

func (r InstallBuildProcess) Run(modulesDir, cacheDir, workingDir, npmrcPath string, launch bool) error {
	err := os.Mkdir(filepath.Join(modulesDir, "node_modules"), os.ModePerm)
	if err != nil {
		return err
	}

	// Set up pnpm store directory for caching
	storeDir := filepath.Join(cacheDir, "store")
	err = os.MkdirAll(storeDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create pnpm store directory: %w", err)
	}

	environment := os.Environ()
	// Set pnpm store-dir via environment variable
	environment = append(environment, fmt.Sprintf("PNPM_HOME=%s", cacheDir))

	if value, ok := r.environment.Lookup("NPM_CONFIG_LOGLEVEL"); ok {
		environment = append(environment, fmt.Sprintf("NPM_CONFIG_LOGLEVEL=%s", value))
	}

	if npmrcPath != "" {
		environment = append(environment, fmt.Sprintf("NPM_CONFIG_GLOBALCONFIG=%s", npmrcPath))
	}

	if !launch {
		environment = append(environment, "NODE_ENV=development")
	}

	// Use pnpm install with frozen-lockfile and store-dir
	// Use --reporter=append-only for better CI logging
	args := []string{"install", "--frozen-lockfile", "--store-dir", storeDir, "--reporter=append-only"}
	r.logger.Subprocess("Running 'pnpm %s'", strings.Join(args, " "))

	err = r.executable.Execute(pexec.Execution{
		Args:   args,
		Dir:    workingDir,
		Stdout: r.logger.ActionWriter,
		Stderr: r.logger.ActionWriter,
		Env:    environment,
	})
	if err != nil {
		return fmt.Errorf("pnpm install failed: %w", err)
	}

	_, err = os.Stat(filepath.Join(workingDir, "node_modules"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("unable to stat node_modules in working directory: %w", err)
	}

	err = fs.Move(filepath.Join(workingDir, "node_modules"), filepath.Join(modulesDir, "node_modules"))
	if err != nil {
		return err
	}

	err = os.Symlink(filepath.Join(modulesDir, "node_modules"), filepath.Join(workingDir, "node_modules"))
	if err != nil {
		return err
	}

	return nil
}
