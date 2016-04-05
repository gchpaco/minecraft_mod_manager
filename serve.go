package main

import (
	"flag"
	"fmt"
	"github.com/gchpaco/minecraft_mod_manager/curseforge"
	"github.com/gchpaco/minecraft_mod_manager/types"
	"github.com/jinzhu/gorm"
	"html/template"
	"log"
	"net/http"
	"strconv"
)

var listenPort = flag.Int64("port", 8080, "port to listen on")
var db *gorm.DB
var templates = template.Must(template.ParseFiles("html/root.html", "html/mod.html"))

func serveHTTP(dbHandle *gorm.DB) {
	db = dbHandle
	http.HandleFunc("/mods/", specificMod)
	http.HandleFunc("/update/", updateMod)
	http.HandleFunc("/", root)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *listenPort), nil))
}

type modList struct {
	Mods []types.Mod
}

func updateMod(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Path[len("/update/"):]
	var mod types.Mod
	db.First(&mod, types.Mod{Name: name})
	if mod.Name != name {
		http.Error(w, "Don't know anything about "+name, http.StatusNotFound)
		return
	}
	var err error
	var page int
	pageStr := r.URL.Query().Get("page")
	if pageStr == "" {
		err = curseforge.UpdateMod(db, &mod)
		page = 1
	} else {
		page, err = strconv.Atoi(pageStr)
		if err == nil {
			err = curseforge.UpdateModPage(db, &mod, page)
		} else {
			err = curseforge.UpdateMod(db, &mod)
			page = 1
		}
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/mods/%s?page=%d", name, page), http.StatusFound)
}

type specifiedMod struct {
	Mod            *types.Mod
	Page, NextPage int
}

func specificMod(w http.ResponseWriter, r *http.Request) {
	var mod types.Mod
	db.First(&mod, types.Mod{Name: r.URL.Path[len("/mods/"):]})
	db.Model(&mod).Related(&mod.Releases)

	var specifiedMod specifiedMod
	var err error

	specifiedMod.Mod = &mod
	pageStr := r.URL.Query().Get("page")
	if pageStr == "" {
		specifiedMod.Page = 1
	} else {
		specifiedMod.Page, err = strconv.Atoi(pageStr)
		if err != nil {
			specifiedMod.Page = 1
		}
	}
	specifiedMod.NextPage = specifiedMod.Page + 1

	err = templates.ExecuteTemplate(w, "mod.html", specifiedMod)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func root(w http.ResponseWriter, r *http.Request) {
	var mods modList
	db.Find(&mods.Mods)

	err := templates.ExecuteTemplate(w, "root.html", mods)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
