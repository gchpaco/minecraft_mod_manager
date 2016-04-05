package types

import (
	"encoding/hex"
	"fmt"
	"net/url"
	"strconv"
)

// GetProjectURL constructs the URL for the Curseforge project itself.
func (mod *Mod) GetProjectURL() *url.URL {
	trueURL := new(url.URL)
	trueURL.Scheme = "https"
	trueURL.Host = "minecraft.curseforge.com"
	trueURL.Path = fmt.Sprintf("/projects/%s", mod.Name)
	return trueURL
}

// GetReleasesURL constructs the URL for the Curseforge releases page.
func (mod *Mod) GetReleasesURL() *url.URL {
	trueURL := new(url.URL)
	trueURL.Scheme = "https"
	trueURL.Host = "minecraft.curseforge.com"
	trueURL.Path = fmt.Sprintf("/projects/%s/files", mod.Name)
	return trueURL
}

// GetReleasesPageURL constructs the URL for the Curseforge releases
// on the given page.
func (mod *Mod) GetReleasesPageURL(page int) *url.URL {
	baseURL := mod.GetReleasesURL()
	q := baseURL.Query()
	q.Set("page", strconv.Itoa(page))
	baseURL.RawQuery = q.Encode()
	return baseURL
}

// GetReleaseURL returns a URL for the release details page on Curseforge.
func (release *Release) GetReleaseURL() *url.URL {
	trueURL := new(url.URL)
	trueURL.Scheme = "https"
	trueURL.Host = "minecraft.curseforge.com"
	trueURL.Path = fmt.Sprintf("/projects/%s/files/%s", release.Mod.Name, release.CurseForgeID)
	return trueURL
}

// MD5 formats the release's MD5sum field as a hex string.
func (release *Release) MD5() string {
	return hex.EncodeToString(release.MD5sum)
}
