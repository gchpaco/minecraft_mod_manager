package curseforge

import (
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/gchpaco/minecraft_mod_manager/types"
	xmlpath "gopkg.in/xmlpath.v2"
	"io"
	"path"
	"strconv"
	"time"
)

var (
	// ErrBadParse is triggered if there is some error parsing the
	// Curseforge results.  This is difficult to avoid as we are
	// forced to screen scrape.
	ErrBadParse = errors.New("Couldn't understand curseforge page")

	eachFilePath = xmlpath.MustCompile("//tr[contains(@class,\"project-file-list-item\")]")
	maturityPath = xmlpath.MustCompile("./td[contains(@class,\"project-file-release-type\")]/div/@title")
	datePath     = xmlpath.MustCompile("./td[contains(@class,\"project-file-date-uploaded\")]/abbr/@data-epoch")
	filenamePath = xmlpath.MustCompile("./td[contains(@class,\"project-file-name\")]//a[contains(@class,\"overflow-tip\")]/text()")
	downloadPath = xmlpath.MustCompile("./td[contains(@class,\"project-file-name\")]//a[contains(@class,\"overflow-tip\")]/@href")
	versionPath  = xmlpath.MustCompile("./td[contains(@class,\"project-file-game-version\")]//span[contains(@class,\"version-label\")]/text()")
	md5path      = xmlpath.MustCompile("//div[contains(@class,\"details-info\")]//span[contains(@class,\"md5\")]/text()")
)

func parseReleases(mod *types.Mod, body io.ReadCloser) (releases []*types.Release, err error) {
	doc, e := xmlpath.ParseHTML(body)
	if e != nil {
		return nil, e
	}

	iter := eachFilePath.Iter(doc)
	for iter.Next() {
		file := iter.Node()

		var release types.Release
		var ok bool
		release.ModID = mod.ID
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

		date, ok := datePath.String(file)
		if !ok {
			err = ErrBadParse
			return
		}
		epochSecs, e := strconv.ParseInt(date, 10, 64)
		if e != nil {
			err = ErrBadParse
			return
		}
		release.DateUploaded = time.Unix(epochSecs, 0)

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

func parseMD5(body io.ReadCloser) ([]byte, error) {
	doc, err := xmlpath.ParseHTML(body)
	if err != nil {
		return []byte{}, err
	}

	md5, ok := md5path.String(doc)
	if !ok {
		return []byte{}, ErrBadParse
	}
	binary, err := hex.DecodeString(md5)
	if err != nil {
		return []byte{}, err
	}
	return binary, nil

}
