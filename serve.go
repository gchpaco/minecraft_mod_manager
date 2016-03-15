package main

import (
	"database/sql"
	"encoding/hex"
	"flag"
	"fmt"
	"github.com/gchpaco/minecraft_mod_manager/curseforge"
	"html/template"
	"log"
	"net/http"
)

var listenPort = flag.Int64("port", 8080, "port to listen on")
var dbHandle *sql.DB
var templates = template.Must(template.ParseFiles("root.html", "mod.html"))

func serveHTTP(db *sql.DB) {
	dbHandle = db
	http.HandleFunc("/mods/", specificMod)
	http.HandleFunc("/update/", updateMod)
	http.HandleFunc("/", root)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *listenPort), nil))
}

type modList struct {
	Mods []*curseforge.Mod
}

func updateMod(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Path[len("/update/"):]
	err := loadMod(dbHandle, name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/mods/"+name, http.StatusFound)
}
func specificMod(w http.ResponseWriter, r *http.Request) {
	var mod curseforge.Mod
	mod.Name = r.URL.Path[len("/mods/"):]
	rows, err := dbHandle.Query(`SELECT cfid, filename, maturity, version, md5sum FROM releases WHERE mod=$1`, mod.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var cfid, filename, maturity, version, md5sum string
		if err := rows.Scan(&cfid, &filename, &maturity, &version, &md5sum); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		md5, err := hex.DecodeString(md5sum)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		mod.Releases = append(mod.Releases, &curseforge.Release{
			CurseForgeID: cfid,
			Mod:          &mod,
			Maturity:     maturity,
			Version:      version,
			Filename:     filename,
			MD5sum:       md5,
		})
	}
	if err := rows.Err(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = templates.ExecuteTemplate(w, "mod.html", mod)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func root(w http.ResponseWriter, r *http.Request) {
	var mods modList
	rows, err := dbHandle.Query(`SELECT name FROM mods;`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var mod string
		if err := rows.Scan(&mod); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		mods.Mods = append(mods.Mods, &curseforge.Mod{Name: mod})
	}
	if err := rows.Err(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = templates.ExecuteTemplate(w, "root.html", mods)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
