package main

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"flag"
	"github.com/gchpaco/minecraft_mod_manager/curseforge"
	"github.com/gchpaco/minecraft_mod_manager/types"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
)

var dbFile = flag.String("db", "db.sqlite", "database to reference against")

func main() {
	flag.Parse()

	db, err := gorm.Open("sqlite3", *dbFile)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Fatal(err)
		}
	}()
	db.AutoMigrate(&types.Mod{}, &types.Release{})

	if flag.NArg() < 1 {
		log.Fatal("Need a command.")
	}

	switch flag.Arg(0) {
	case "add":
		for _, name := range flag.Args()[1:] {
			log.Printf("Adding mod: %s\n", name)
			var mod types.Mod
			db.FirstOrCreate(&mod, types.Mod{Name: name})

			err = curseforge.UpdateMod(db, &mod)
			if err != nil {
				log.Println(err)
				continue
			}
		}
	case "update":
		var mods []types.Mod
		db.Find(&mods)

		for _, mod := range mods {
			log.Printf("Updating mod: %s\n", mod.Name)
			err = curseforge.UpdateMod(db, &mod)
			if err != nil {
				log.Println(err)
				continue
			}
		}
	case "serve":
		serveHTTP(db)
	case "scan":
		for _, directory := range flag.Args()[1:] {
			log.Println("Scanning", directory)

			files, err := ioutil.ReadDir(directory)
			if err != nil {
				log.Println(err)
				continue
			}

			for _, file := range files {
				// Mac nonsense
				if file.Name() == ".DS_Store" {
					continue
				}
				if !file.IsDir() {
					sum, err := md5sum(path.Join(directory, file.Name()))
					if err != nil {
						log.Println(err)
						continue
					}
					var release types.Release
					db.First(&release, types.Release{MD5sum: sum})
					if release.MD5sum == nil {
						log.Println("No match for", file.Name())
						continue
					}
				}
			}
		}
		/*	case "orm-migrate":
				if err := db.Close(); err != nil {
					log.Fatal(err)
				}
				database, err := gorm.Open("sqlite3", *dbFile)
				if err != nil {
					log.Fatal(err)
				}
				defer func() {
					if err := database.Close(); err != nil {
						log.Fatal(err)
					}
				}()
				database.AutoMigrate(&ormdb.Mod{}, &ormdb.Release{})

				var oldMods []struct {
					Name string
				}
				var oldReleases []struct {
					Cfid     uint
					Mod      string
					Maturity string
					Filename string
					Version  string
					Md5sum   string
					Date     string
				}
				database.Raw(`SELECT name FROM old_mods`).Scan(&oldMods)
				for _, oldMod := range oldMods {
					var mod ormdb.Mod
					database.FirstOrCreate(&mod, ormdb.Mod{Name: oldMod.Name})

					database.Debug().Raw(`SELECT cfid, mod, maturity, filename, version, md5sum, date FROM old_releases WHERE mod=?`, mod.Name).Scan(&oldReleases)
					for _, oldRelease := range oldReleases {
						md5, err := hex.DecodeString(oldRelease.Md5sum)
						if err != nil {
							log.Println(mod.Name, oldRelease.Filename, err)
							continue
						}
						date, err := time.Parse("2006-01-02 15:04:05-07:00", oldRelease.Date)
						if err != nil {
							log.Println(mod.Name, oldRelease.Filename, err)
							continue
						}
						cfid := fmt.Sprintf("%d", oldRelease.Cfid)
						release := ormdb.Release{
							ModID:        mod.ID,
							Maturity:     oldRelease.Maturity,
							Filename:     oldRelease.Filename,
							Version:      oldRelease.Version,
							CurseForgeID: cfid,
							DateUploaded: date,
							MD5sum:       md5,
						}
						database.Debug().FirstOrCreate(&release, ormdb.Release{ModID: mod.ID, CurseForgeID: cfid})
					}
				}
			case "orm-dump":
				if err := db.Close(); err != nil {
					log.Fatal(err)
				}
				database, err := gorm.Open("sqlite3", *dbFile)
				database.AutoMigrate(&ormdb.Mod{}, &ormdb.Release{})
				if err != nil {
					log.Fatal(err)
				}
				defer func() {
					if err := database.Close(); err != nil {
						log.Fatal(err)
					}
				}()

				var mods []ormdb.Mod
				database.Debug().Preload("Releases").Find(&mods)
				for _, mod := range mods {
					log.Println(mod.Name)
					var releases []ormdb.Release
					database.Debug().Model(&mod).Related(&releases)
					for _, release := range releases {
						log.Println(release.Filename)
					}
				}
		*/
	case "find-updates":
		type bests struct {
			ID       uint
			BestID   uint
			ModID    uint
			Filename string
			MD5sum   []byte
			Version  string
		}
		bestVersion := make(map[string]*bests)
		rows, err := db.Table("releases").Raw(`
SELECT releases.id, releases.md5sum, best.id AS best_id, releases.mod_id, best.filename, releases.version
FROM releases
JOIN releases as best ON releases.mod_id=best.mod_id AND releases.version=best.version
WHERE best.id=(SELECT r.id FROM releases AS r
               WHERE r.mod_id=releases.mod_id AND r.version=releases.version
               ORDER BY date_uploaded DESC LIMIT 1);
`).Rows()
		if err != nil {
			log.Fatal(err)
		}
		defer rows.Close()
		for rows.Next() {
			var id, bestid, modid uint
			var md5sum []byte
			var filename, version sql.NullString
			if err := rows.Scan(&id, &md5sum, &bestid, &modid, &filename, &version); err != nil {
				log.Println(err)
				continue
			}
			if !filename.Valid || !version.Valid {
				log.Println("Weird nulls")
				continue
			}
			bestVersion[hex.EncodeToString(md5sum)] = &bests{
				ID:       id,
				BestID:   bestid,
				ModID:    modid,
				Filename: filename.String,
				MD5sum:   md5sum,
				Version:  version.String,
			}
		}
		if err := rows.Err(); err != nil {
			log.Fatal(err)
		}

		for _, directory := range flag.Args()[1:] {
			log.Println("Scanning", directory)

			files, err := ioutil.ReadDir(directory)
			if err != nil {
				log.Println(err)
				continue
			}

			for _, file := range files {
				// Mac nonsense
				if file.Name() == ".DS_Store" {
					continue
				}
				if !file.IsDir() {
					sum, err := md5sum(path.Join(directory, file.Name()))
					if err != nil {
						log.Println(err)
						continue
					}
					best, ok := bestVersion[hex.EncodeToString(sum)]
					if ok && best.ID != best.BestID {
						var bestRelease types.Release
						db.First(&bestRelease, best.BestID)
						log.Println("Better version of", file.Name(), "available:", best.Filename, bestRelease.GetReleaseURL())
					}
				}
			}
		}
	}
}

func md5sum(filename string) ([]byte, error) {
	hash := md5.New()

	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	data := make([]byte, 128)
	for {
		count, err := file.Read(data)
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		hash.Write(data[:count])
	}
	return hash.Sum(nil), nil
}
