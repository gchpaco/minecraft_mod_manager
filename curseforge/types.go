package curseforge

import "net/url"
import "time"

// Mod is a capsule description of a mod on CurseForge.
type Mod struct {
	Name     string
	Releases []*Release
}

// Release is a specific release of a mod.
type Release struct {
	Mod                         *Mod
	Maturity, Filename, Version string
	CurseForgeID                string
	DownloadURL                 *url.URL
	DateUploaded                time.Time
	MD5sum                      []byte
}
