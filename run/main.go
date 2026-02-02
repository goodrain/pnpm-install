package main

import (
	"log"
	"os"
	"path/filepath"

	pnpminstall "github.com/goodrain/pnpm-install"

	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/chronos"
	"github.com/paketo-buildpacks/packit/v2/draft"
	"github.com/paketo-buildpacks/packit/v2/fs"
	"github.com/paketo-buildpacks/packit/v2/pexec"
	"github.com/paketo-buildpacks/packit/v2/sbom"
	"github.com/paketo-buildpacks/packit/v2/scribe"
	"github.com/paketo-buildpacks/packit/v2/servicebindings"
)

type SBOMGenerator struct{}

func (s SBOMGenerator) Generate(path string) (sbom.SBOM, error) {
	return sbom.Generate(path)
}

func main() {
	environment, err := pnpminstall.ParseEnvironment(filepath.Join(os.Getenv("CNB_BUILDPACK_DIR"), "buildpack.toml"), os.Environ())
	if err != nil {
		log.Fatal(err)
	}

	logLevel, _ := environment.Lookup("BP_LOG_LEVEL")
	globalConfigPath, _ := environment.Lookup("NPM_CONFIG_GLOBALCONFIG")

	emitter := scribe.NewEmitter(os.Stdout).WithLevel(logLevel)
	logger := scribe.NewLogger(os.Stdout).WithLevel(logLevel)

	pnpm := pexec.NewExecutable("pnpm")
	checksumCalculator := fs.NewChecksumCalculator()
	linker := pnpminstall.NewLinker(os.TempDir())

	packit.Run(
		pnpminstall.Detect(),
		pnpminstall.Build(
			draft.NewPlanner(),
			pnpminstall.NewPackageManagerConfigurationManager(
				servicebindings.NewResolver(),
				emitter,
				globalConfigPath,
			),
			pnpminstall.NewBuildProcessResolver(
				logger,
				pnpminstall.NewRebuildBuildProcess(pnpm, checksumCalculator, environment, logger),
				pnpminstall.NewInstallBuildProcess(pnpm, environment, logger),
				pnpminstall.NewCIBuildProcess(pnpm, checksumCalculator, environment, logger),
			),
			pnpminstall.NewPruneBuildProcess(
				pnpm,
				environment,
				logger,
			),
			chronos.DefaultClock,
			emitter,
			SBOMGenerator{},
			linker,
			environment,
			pnpminstall.NewLinkedModuleResolver(linker),
		),
	)
}
