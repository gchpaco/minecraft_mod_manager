package curseforge

import (
	"github.com/gchpaco/minecraft_mod_manager/types"
	"github.com/jinzhu/gorm"
)

// UpdateMod retrieves information for the given mod from Curseforge.
func UpdateMod(db *gorm.DB, mod *types.Mod) error {
	releases, err := getReleases(mod)
	if err != nil {
		return err
	}
	for _, release := range releases {
		db.FirstOrCreate(&release, types.Release{CurseForgeID: release.CurseForgeID})
	}
	return nil
}

// UpdateModPage retrieves information for the given mod from
// Curseforge, reading from page N of the release list.
func UpdateModPage(db *gorm.DB, mod *types.Mod, page int) error {
	releases, err := getReleasesPage(mod, page)
	if err != nil {
		return err
	}
	for _, release := range releases {
		db.FirstOrCreate(release, types.Release{CurseForgeID: release.CurseForgeID})
		if len(release.MD5sum) == 0 {
			err := fetchMD5sum(release)
			if err != nil {
				return err
			}
			db.Save(&release)
		}
	}
	return nil
}
