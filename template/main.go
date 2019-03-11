package main

import (
	"log"
	"net/http"

	"github.com/jmu0/components"
)

var staticDir = "static"
var listenAddr = ":8282"

func main() {
	mx := http.NewServeMux()
	// mx.HandleFunc("/favicon.ico", http.FileServer(http.Dir(staticDir)).ServeHTTP)
	mx.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-control", "max-age=86400")
		http.FileServer(http.Dir(staticDir)).ServeHTTP(w, r)
		// http.ServeFile(w, r, staticDir+"/favicon.ico")
	})
	mx.HandleFunc("/"+staticDir+"/", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Serving:", r.URL.Path)
		w.Header().Set("Cache-control", "max-age=90")
		http.FileServer(http.Dir("./")).ServeHTTP(w, r)
	})
	var app = components.App{
		ConfigFile: "app.json",
		Mux:        mx,
	}
	err := app.Init()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("DEBUG:", app.Debug)
	log.Println("Listening on port", listenAddr)
	log.Fatal(http.ListenAndServe(listenAddr, mx))
}
