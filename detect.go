package pnpminstall

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/paketo-buildpacks/libnodejs"
	"github.com/paketo-buildpacks/packit/v2"
)

type BuildPlanMetadata struct {
	Version       string `toml:"version"`
	VersionSource string `toml:"version-source"`
	Build         bool   `toml:"build"`
	Launch        bool   `toml:"launch"`
}

//go:generate faux --interface VersionParser --output fakes/version_parser.go
type VersionParser interface {
	ParseVersion(path string) (version string, err error)
}

func Detect() packit.DetectFunc {
	return func(context packit.DetectContext) (packit.DetectResult, error) {
		var requirements []packit.BuildPlanRequirement

		projectPath, err := libnodejs.FindProjectPath(context.WorkingDir)
		if err != nil {
			return packit.DetectResult{}, err
		}

		// Check if pnpm-lock.yaml exists
		lockfilePath := filepath.Join(projectPath, PnpmLockfile)
		if _, err := os.Stat(lockfilePath); os.IsNotExist(err) {
			return packit.DetectResult{}, packit.Fail.WithMessage("no '%s' found in project path %s", PnpmLockfile, projectPath)
		}

		pkg, err := libnodejs.ParsePackageJSON(projectPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return packit.DetectResult{}, packit.Fail.WithMessage("no 'package.json' found in project path %s", filepath.Join(projectPath))
			}

			return packit.DetectResult{}, err
		}
		version := pkg.GetVersion()

		nodeDependency := packit.BuildPlanRequirement{
			Name: Node,
			Metadata: BuildPlanMetadata{
				Build: true,
			},
		}

		if version != "" {
			nodeDependency.Metadata = BuildPlanMetadata{
				Version:       version,
				VersionSource: "package.json",
				Build:         true,
			}
		}

		requirements = append(requirements, nodeDependency)

		bpPnpmIncludeBuildPython, bpPnpmIncludeBuildPythonExists := os.LookupEnv("BP_PNPM_INCLUDE_BUILD_PYTHON")

		installPython := false
		if bpPnpmIncludeBuildPythonExists && (bpPnpmIncludeBuildPython == "" || bpPnpmIncludeBuildPython == "true") {
			installPython = true
		} else if bpPnpmIncludeBuildPythonExists && bpPnpmIncludeBuildPython == "false" {
			installPython = false
		}

		if installPython {
			requirements = append(requirements, packit.BuildPlanRequirement{
				Name: Cpython,
				Metadata: BuildPlanMetadata{
					Build:  true,
					Launch: false,
				},
			})
		}

		// Note: pnpm will be installed via npm if BP_PNPM_VERSION is set,
		// or we assume pnpm is available in the build environment

		return packit.DetectResult{
			Plan: packit.BuildPlan{
				Provides: []packit.BuildPlanProvision{
					{Name: NodeModules},
				},
				Requires: requirements,
			},
		}, nil
	}
}
