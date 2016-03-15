package main

import "github.com/gchpaco/minecraft_mod_manager/curseforge"
import "flag"
import "log"
import "encoding/hex"

func main() {
	flag.Parse()

	mod, err := curseforge.FetchMod(flag.Arg(0))
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Mod: %s\n", mod.Name)
	for _, release := range mod.Releases {
		log.Printf("Release: %s for %s\n", release.Filename, release.Version)
		log.Printf("\tMaturity: %s\n", release.Maturity)
		log.Printf("\tCF ID: %s\n", release.CurseForgeID)
		if err := release.FetchMD5Sum(); err != nil {
			log.Fatal(err)
		}
		log.Printf("\tMD5: %s\n", hex.EncodeToString(release.MD5sum))
	}
}
