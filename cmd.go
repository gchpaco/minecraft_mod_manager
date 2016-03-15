package main

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"flag"
	"github.com/gchpaco/minecraft_mod_manager/curseforge"
	_ "github.com/mattn/go-sqlite3"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"time"
)

var dbFile = flag.String("db", "db.sqlite", "database to reference against")

func main() {
	flag.Parse()

	db, err := sql.Open("sqlite3", *dbFile)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	err = setupSchema(db)
	if err != nil {
		log.Fatal(err)
	}

	if flag.NArg() < 1 {
		log.Fatal("Need a command.")
	}

	switch flag.Arg(0) {
	case "add":
		for _, mod := range flag.Args()[1:] {
			log.Printf("Adding mod: %s\n", mod)
			err = loadMod(db, mod)
			if err != nil {
				log.Println(err)
				continue
			}
		}
	case "update":
		var mods []string

		rows, err := db.Query(`SELECT name FROM mods;`)
		if err != nil {
			log.Fatal(err)
		}
		defer rows.Close()
		for rows.Next() {
			var mod string
			if err := rows.Scan(&mod); err != nil {
				log.Println(err)
				continue
			}
			mods = append(mods, mod)
		}
		if err := rows.Err(); err != nil {
			log.Fatal(err)
		}

		for _, mod := range mods {
			log.Printf("Updating mod: %s\n", mod)
			err = loadMod(db, mod)
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
					rows, err := db.Query(`SELECT cfid, mod, filename FROM releases WHERE md5sum=$1`, hex.EncodeToString(sum))
					if err != nil {
						log.Println(err)
						continue
					}
					defer rows.Close()

					matched := false
					for rows.Next() {
						var cfid, mod, filename string
						if err := rows.Scan(&cfid, &mod, &filename); err != nil {
							log.Println(err)
							break
						}
						matched = true
					}
					if err := rows.Err(); err != nil {
						log.Println(err)
						continue
					}
					if !matched {
						log.Println("No match for", file.Name())
					}
				}
			}
		}
	case "find-updates":
		// assume 1.7.10 for now.
		bestVersion := make(map[string]*curseforge.Release)
		rows, err := db.Query(`
SELECT mods.name, releases.cfid, releases.maturity, releases.filename, releases.version, releases.date, releases.md5sum
FROM mods, releases
WHERE releases.cfid=(SELECT releases.cfid FROM releases WHERE releases.mod=mods.name AND releases.version='1.7.10' ORDER BY date DESC LIMIT 1);
`)
		if err != nil {
			log.Fatal(err)
		}
		defer rows.Close()
		for rows.Next() {
			var modname, cfid, maturity, filename, version, datestr, md5sum sql.NullString
			if err := rows.Scan(&modname, &cfid, &maturity, &filename, &version, &datestr, &md5sum); err != nil {
				log.Println(err)
				continue
			}
			if !modname.Valid || !cfid.Valid || !filename.Valid {
				log.Println("Weird nulls")
				continue
			}
			date, err := time.Parse("2006-01-02 15:04:05-07:00", datestr.String)
			if err != nil {
				log.Println(modname.String, err)
				continue
			}
			md5, err := hex.DecodeString(md5sum.String)
			if err != nil {
				log.Println(err)
				continue
			}
			mod := &curseforge.Mod{Name: modname.String}
			release := &curseforge.Release{
				Mod:          mod,
				CurseForgeID: cfid.String,
				Maturity:     maturity.String,
				Filename:     filename.String,
				Version:      version.String,
				DateUploaded: date,
				MD5sum:       md5,
			}
			bestVersion[modname.String] = release
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
					rows, err := db.Query(`SELECT cfid, mod, filename FROM releases WHERE md5sum=$1`, hex.EncodeToString(sum))
					if err != nil {
						log.Println(err)
						continue
					}
					defer rows.Close()

					for rows.Next() {
						var cfid, mod, filename string
						if err := rows.Scan(&cfid, &mod, &filename); err != nil {
							log.Println(err)
							break
						}
						best, ok := bestVersion[mod]
						if ok && cfid != best.CurseForgeID {
							log.Println("Better version of", file.Name(), "available:", best.Filename, bestVersion[mod].GetReleaseURL())
						}
					}
					if err := rows.Err(); err != nil {
						log.Println(err)
						continue
					}
				}
			}
		}
	}
}

func setupSchema(db *sql.DB) error {
	_, err := db.Exec(`
CREATE TABLE IF NOT EXISTS mods(name TEXT PRIMARY KEY);
CREATE TABLE IF NOT EXISTS releases(cfid INT PRIMARY KEY, mod TEXT, maturity, filename TEXT, version TEXT, date, md5sum TEXT);
CREATE INDEX IF NOT EXISTS releases_date ON releases(date);
CREATE INDEX IF NOT EXISTS releases_date_version ON releases(date, version);
CREATE INDEX IF NOT EXISTS releases_md5sum ON releases(md5sum);
CREATE INDEX IF NOT EXISTS releases_mod ON releases(mod);
`)
	if err != nil {
		return err
	}
	return nil
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

func loadMod(db *sql.DB, name string) error {
	mod, err := curseforge.FetchMod(name)
	if err != nil {
		return err
	}

	return updateForMod(db, mod)
}

func loadModPage(db *sql.DB, name string, page int) error {
	mod, err := curseforge.FetchModPage(name, page)
	if err != nil {
		return err
	}

	return updateForMod(db, mod)
}

func updateForMod(db *sql.DB, mod *curseforge.Mod) error {
	var err error
	_, err = db.Exec(`INSERT OR IGNORE INTO mods(name) VALUES ($1);`, mod.Name)
	if err != nil {
		return err
	}
	for _, release := range mod.Releases {
		tx, err := db.Begin()
		if err != nil {
			return err
		}

		var count sql.NullInt64
		result := tx.QueryRow(`SELECT count(*) FROM releases WHERE cfid=$1`, release.CurseForgeID)
		err = result.Scan(&count)
		if err != nil {
			return err
		}
		if !count.Valid || count.Int64 != 1 {
			if err := release.FetchMD5Sum(); err != nil {
				return err
			}
			_, err = tx.Exec(`
INSERT INTO
releases(cfid, mod, maturity, filename, version, date, md5sum)
VALUES ($1, $2, $3, $4, $5, $6, $7);
`,
				release.CurseForgeID, mod.Name, release.Maturity,
				release.Filename, release.Version, release.DateUploaded,
				hex.EncodeToString(release.MD5sum))
			if err != nil {
				return err
			}
		} else {
			_, err := tx.Exec(`
UPDATE releases SET
mod=$1, maturity=$2, filename=$3, version=$4, date=$5
WHERE cfid=$6;
`,
				mod.Name, release.Maturity,
				release.Filename, release.Version, release.DateUploaded,
				release.CurseForgeID)
			if err != nil {
				return err
			}
			var md5 sql.NullString
			result := db.QueryRow(`SELECT md5sum FROM releases WHERE cfid=$1;`, release.CurseForgeID)
			err = result.Scan(&md5)
			if err != nil {
				return err
			}
			if !md5.Valid || md5.String == "" {
				log.Println("Fetching md5")
				if err := release.FetchMD5Sum(); err != nil {
					return err
				}
				_, err = db.Exec(`UPDATE releases SET md5sum=$2 WHERE cfid=$1`,
					release.CurseForgeID, release.MD5())
				if err != nil {
					return err
				}
			}
		}
		tx.Commit()
	}
	return nil
}
