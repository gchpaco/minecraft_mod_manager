package main

import (
	"database/sql"
	"encoding/hex"
	"flag"
	"github.com/gchpaco/minecraft_mod_manager/curseforge"
	_ "github.com/mattn/go-sqlite3"
	"log"
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

	var mods []string
	mods = flag.Args()

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
}

func setupSchema(db *sql.DB) error {
	_, err := db.Exec(`
CREATE TABLE IF NOT EXISTS mods(name TEXT PRIMARY KEY);
CREATE TABLE IF NOT EXISTS releases(cfid INT PRIMARY KEY, mod TEXT, maturity, filename TEXT, version TEXT, md5sum TEXT);
CREATE INDEX IF NOT EXISTS releases_md5sum ON releases(md5sum);
CREATE INDEX IF NOT EXISTS releases_mod ON releases(mod);
`)
	if err != nil {
		return err
	}
	return nil
}

func loadMod(db *sql.DB, name string) error {
	mod, err := curseforge.FetchMod(name)
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
releases(cfid, mod, maturity, filename, version, md5sum)
VALUES ($1, $2, $3, $4, $5, $6);
`,
				release.CurseForgeID, mod.Name, release.Maturity,
				release.Filename, release.Version,
				hex.EncodeToString(release.MD5sum))
			if err != nil {
				return err
			}
		} else {
			_, err := tx.Exec(`
UPDATE releases SET
mod=$2, maturity=$3, filename=$4, version=$5
WHERE cfid=$1;
`)
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
					release.CurseForgeID, hex.EncodeToString(release.MD5sum))
				if err != nil {
					return err
				}
			}
		}
		tx.Commit()
	}
	return nil
}
