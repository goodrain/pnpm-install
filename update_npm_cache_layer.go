package pnpminstall

import (
	"os"
	"path/filepath"

	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/fs"
	"github.com/paketo-buildpacks/packit/v2/scribe"
)

func UpdatePnpmCacheLayer(logger scribe.Emitter, workingDir string, cacheLayer packit.Layer) (packit.Layer, error) {
	pnpmCachePath := filepath.Join(workingDir, "pnpm-store")
	sum, err := fs.NewChecksumCalculator().Sum(pnpmCachePath)
	if err != nil {
		return packit.Layer{}, err
	}

	cacheSha, ok := cacheLayer.Metadata["cache_sha"].(string)
	if !ok || sum != cacheSha {
		if err != nil {
			return packit.Layer{}, err
		}

		err = fs.Move(pnpmCachePath, cacheLayer.Path)
		if err != nil {
			return packit.Layer{}, err
		}

		cacheLayer.Metadata = map[string]interface{}{
			"cache_sha": sum,
		}
	} else {
		logger.Process("Reusing cached layer %s", cacheLayer.Path)
		err = os.RemoveAll(pnpmCachePath)
		if err != nil {
			return packit.Layer{}, err
		}
	}

	return cacheLayer, nil
}
