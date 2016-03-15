package curseforge

import "net/url"

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
	MD5sum                      []byte
}
