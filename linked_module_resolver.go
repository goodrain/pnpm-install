package pnpminstall

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/paketo-buildpacks/packit/v2/fs"
	"gopkg.in/yaml.v3"
)

// PnpmLockfileData represents the structure of pnpm-lock.yaml
type PnpmLockfileData struct {
	Importers map[string]struct {
		Dependencies         map[string]interface{} `yaml:"dependencies"`
		DevDependencies      map[string]interface{} `yaml:"devDependencies"`
		OptionalDependencies map[string]interface{} `yaml:"optionalDependencies"`
	} `yaml:"importers"`
	Packages map[string]struct {
		Resolution struct {
			Directory string `yaml:"directory"`
		} `yaml:"resolution"`
	} `yaml:"packages"`
}

type Lockfile struct {
	Packages map[string]struct {
		Resolved string `json:"resolved"`
		Link     bool   `json:"link"`
	} `json:"packages"`
}

type LinkedModuleResolver struct {
	linker Symlinker
}

func NewLinkedModuleResolver(linker Symlinker) LinkedModuleResolver {
	return LinkedModuleResolver{
		linker: linker,
	}
}

func (r LinkedModuleResolver) ParseLockfile(lockfilePath string) (lockfile Lockfile, err error) {
	file, err := os.Open(lockfilePath)
	if err != nil {
		return Lockfile{}, fmt.Errorf(`failed to open "%s": %w`, PnpmLockfile, err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			if err == nil {
				err = fmt.Errorf(`failed to close "%s": %w`, PnpmLockfile, closeErr)
			}
		}
	}()

	// Parse pnpm-lock.yaml
	var pnpmLockfile PnpmLockfileData
	err = yaml.NewDecoder(file).Decode(&pnpmLockfile)
	if err != nil {
		return Lockfile{}, fmt.Errorf(`failed to parse "%s": %w`, PnpmLockfile, err)
	}

	// Convert to Lockfile format for compatibility
	parsedLockfile := Lockfile{
		Packages: make(map[string]struct {
			Resolved string `json:"resolved"`
			Link     bool   `json:"link"`
		}),
	}

	// pnpm uses link: protocol for local packages
	for pkgName, pkg := range pnpmLockfile.Packages {
		if pkg.Resolution.Directory != "" {
			parsedLockfile.Packages[pkgName] = struct {
				Resolved string `json:"resolved"`
				Link     bool   `json:"link"`
			}{
				Resolved: pkg.Resolution.Directory,
				Link:     true,
			}
		}
	}

	return parsedLockfile, nil
}

func (r LinkedModuleResolver) Copy(lockfilePath, sourceLayerPath, targetLayerPath string) error {

	lockfile, err := r.ParseLockfile(lockfilePath)
	if err != nil {
		// If parsing fails, just return nil (no linked modules to copy)
		return nil
	}

	for _, pkg := range lockfile.Packages {
		if pkg.Link {
			source := filepath.Join(sourceLayerPath, pkg.Resolved)
			destination := filepath.Join(targetLayerPath, pkg.Resolved)

			err = os.MkdirAll(filepath.Dir(destination), os.ModePerm)
			if err != nil {
				return fmt.Errorf("failed to setup linked module directory scaffolding: %w", err)
			}

			err = fs.Copy(source, destination)
			if err != nil {
				return fmt.Errorf("failed to copy linked module directory to layer path: %w", err)
			}
		}
	}

	return nil

}

func (r LinkedModuleResolver) Resolve(lockfilePath, layerPath string) error {

	lockfile, err := r.ParseLockfile(lockfilePath)
	if err != nil {
		// If parsing fails, just return nil (no linked modules to resolve)
		return nil
	}

	dir := filepath.Dir(lockfilePath)
	for _, pkg := range lockfile.Packages {
		if pkg.Link {
			source := filepath.Join(dir, pkg.Resolved)
			destination := filepath.Join(layerPath, pkg.Resolved)

			err = os.MkdirAll(filepath.Dir(destination), os.ModePerm)
			if err != nil {
				return fmt.Errorf("failed to setup linked module directory scaffolding: %w", err)
			}

			err = fs.Copy(source, destination)
			if err != nil {
				return fmt.Errorf("failed to copy linked module directory to layer path: %w", err)
			}

			err = r.linker.WithPath(pkg.Resolved).Link(source, destination)
			if err != nil {
				return fmt.Errorf("failed to symlink linked module directory: %w", err)
			}
		}
	}

	return nil
}
