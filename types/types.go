package types

import "time"
import "github.com/jinzhu/gorm"

// Mod is a capsule description of a mod on CurseForge.
type Mod struct {
	gorm.Model
	Name     string
	Releases []Release
}

// Release is a specific release of a mod.
type Release struct {
	gorm.Model
	ModID                       uint `gorm:"index"`
	Maturity, Filename, Version string
	CurseForgeID                string `gorm:"column:curseforge_id"`
	DateUploaded                time.Time
	MD5sum                      []byte `gorm:"column:md5sum",gorm:"index"`
}
