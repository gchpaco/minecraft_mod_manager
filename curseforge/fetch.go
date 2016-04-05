package curseforge

import (
	"github.com/gchpaco/minecraft_mod_manager/types"
	"net/http"
	"net/url"
)

func getReleases(mod *types.Mod) ([]*types.Release, error) {
	return readReleasesFrom(mod, mod.GetReleasesURL())
}

func getReleasesPage(mod *types.Mod, page int) ([]*types.Release, error) {
	return readReleasesFrom(mod, mod.GetReleasesPageURL(page))
}

func readReleasesFrom(mod *types.Mod, target *url.URL) ([]*types.Release, error) {
	resp, err := http.Get(target.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return parseReleases(mod, resp.Body)
}

// fetchMD5sum fills in the release's MD5sum field.  Since this
// requires a separate fetch, it is not done automatically.
func fetchMD5sum(release *types.Release) error {
	resp, err := http.Get(release.GetReleaseURL().String())
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	sum, err := parseMD5(resp.Body)
	if err != nil {
		return err
	}
	release.MD5sum = sum
	return nil
}
