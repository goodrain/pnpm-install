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

type CIBuildProcess struct {
	executable  Executable
	summer      Summer
	environment EnvironmentConfig
	logger      scribe.Logger
}

func NewCIBuildProcess(executable Executable, summer Summer, environment EnvironmentConfig, logger scribe.Logger) CIBuildProcess {
	return CIBuildProcess{
		executable:  executable,
		summer:      summer,
		environment: environment,
		logger:      logger,
	}
}

func (r CIBuildProcess) ShouldRun(workingDir string, metadata map[string]interface{}, npmrcConfig string) (bool, string, error) {
	cachedNodeVersion, err := cacheExecutableResponse(
		r.executable,
		[]string{"--version"},
		workingDir,
		npmrcConfig,
		r.logger)
	if err != nil {
		return false, "", fmt.Errorf("failed to execute pnpm --version: %w", err)
	}
	defer func() {
		if removeErr := os.Remove(cachedNodeVersion); removeErr != nil {
			r.logger.Subprocess("Warning: failed to remove temporary file %s: %s", cachedNodeVersion, removeErr)
		}
	}()

	sum, err := r.summer.Sum(
		filepath.Join(workingDir, "package.json"),
		filepath.Join(workingDir, PnpmLockfile),
		cachedNodeVersion)
	if err != nil {
		return false, "", err
	}

	cacheSha, ok := metadata["cache_sha"].(string)
	if !ok || sum != cacheSha {
		return true, sum, nil
	}

	return false, "", nil
}

func (r CIBuildProcess) Run(modulesDir, cacheDir, workingDir, npmrcPath string, launch bool) error {
	err := os.MkdirAll(filepath.Join(workingDir, "node_modules"), os.ModePerm)
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

	// Use pnpm install with frozen-lockfile (equivalent to npm ci)
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
