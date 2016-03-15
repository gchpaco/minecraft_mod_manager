package curseforge

import (
	"encoding/hex"
	"errors"
	"fmt"
	xmlpath "gopkg.in/xmlpath.v2"
	"net/http"
	"net/url"
	"path"
)

var (
	// ErrBadParse is triggered if there is some error parsing the
	// Curseforge results.  This is difficult to avoid as we are
	// forced to screen scrape.
	ErrBadParse = errors.New("Couldn't understand curseforge page")

	eachFilePath = xmlpath.MustCompile("//tr[contains(@class,\"project-file-list-item\")]")
	maturityPath = xmlpath.MustCompile("./td[contains(@class,\"project-file-release-type\")]/div/@title")
	filenamePath = xmlpath.MustCompile("./td[contains(@class,\"project-file-name\")]//a[contains(@class,\"overflow-tip\")]/text()")
	downloadPath = xmlpath.MustCompile("./td[contains(@class,\"project-file-name\")]//a[contains(@class,\"overflow-tip\")]/@href")
	versionPath  = xmlpath.MustCompile("./td[contains(@class,\"project-file-game-version\")]//span[contains(@class,\"version-label\")]/text()")
	md5path      = xmlpath.MustCompile("//div[contains(@class,\"details-info\")]//span[contains(@class,\"md5\")]/text()")
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

// FetchMod retrieves information for the given mod from Curseforge.
func FetchMod(project string) (*Mod, error) {
	var mod Mod
	var err error
	mod.Name = project
	mod.Releases, err = mod.fetchReleases()
	if err != nil {
		return nil, err
	}
	return &mod, nil
}

func (mod *Mod) fetchReleases() (releases []*Release, err error) {
	target := mod.GetReleasesURL()
	resp, err := http.Get(target.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	doc, err := xmlpath.ParseHTML(resp.Body)
	if err != nil {
		return nil, err
	}

	iter := eachFilePath.Iter(doc)
	for iter.Next() {
		file := iter.Node()

		var release Release
		var ok bool
		release.Mod = mod
		release.Maturity, ok = maturityPath.String(file)
		if !ok {
			err = ErrBadParse
			return
		}

		release.Filename, ok = filenamePath.String(file)
		if !ok {
			err = ErrBadParse
			return
		}

		partial, ok := downloadPath.String(file)
		if !ok {
			err = ErrBadParse
			return
		}

		localURL, e := url.Parse(partial)
		if e != nil {
			err = fmt.Errorf("Error parsing CF download URLs: %s", e)
			return
		}
		release.DownloadURL = target.ResolveReference(localURL)
		release.CurseForgeID, e = extractCFID(partial)
		if e != nil {
			err = fmt.Errorf("Couldn't extract curseforge ID from %s: %s", partial, e)
			return
		}

		release.Version, ok = versionPath.String(file)
		if !ok {
			err = ErrBadParse
			return
		}

		releases = append(releases, &release)
	}
	return
}

func extractCFID(urlFragment string) (string, error) {
	if ok, err := path.Match("/projects/*/files/*", urlFragment); !ok {
		if err != nil {
			return "", err
		}
		return "", ErrBadParse
	}

	return path.Base(urlFragment), nil
}

// GetReleaseURL returns a URL for the release details page on Curseforge.
func (release *Release) GetReleaseURL() *url.URL {
	trueURL := new(url.URL)
	trueURL.Scheme = "https"
	trueURL.Host = "minecraft.curseforge.com"
	trueURL.Path = fmt.Sprintf("/projects/%s/files/%s", release.Mod.Name, release.CurseForgeID)
	return trueURL
}

// FetchMD5Sum fills in the release's MD5sum field.  Since this
// requires a separate fetch, it is not done automatically.
func (release *Release) FetchMD5Sum() error {
	resp, err := http.Get(release.GetReleaseURL().String())
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	doc, err := xmlpath.ParseHTML(resp.Body)
	if err != nil {
		return err
	}

	md5, ok := md5path.String(doc)
	if !ok {
		return ErrBadParse
	}
	release.MD5sum, err = hex.DecodeString(md5)
	if err != nil {
		return err
	}
	return nil
}
