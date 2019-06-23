package main

import (
	"log"
	"net/http"
	"os"

	"github.com/jmu0/components"
)

var staticDir = "static"

//var listenAddr = ":8282"

func main() {
	var RootPath string
	if len(os.Args) > 1 {
		RootPath = os.Args[1]
	}

	mx := http.NewServeMux()
	mx.HandleFunc("/auth/", handleAuth)
	mx.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-control", "max-age=86400")
		http.FileServer(http.Dir(RootPath+staticDir)).ServeHTTP(w, r)
	})
	mx.HandleFunc("/"+staticDir+"/", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Serving:", r.URL.Path)
		w.Header().Set("Cache-control", "max-age=90")
		if RootPath == "" {
			http.FileServer(http.Dir("./")).ServeHTTP(w, r)
		} else {
			http.FileServer(http.Dir(RootPath)).ServeHTTP(w, r)
		}
	})

	var app = components.App{
		ConfigFile: "app.yml",
		Mux:        mx,
		RootPath:   RootPath,
	}
	err := app.Init()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Debug:", app.Debug)
	log.Println("Listening on port", app.Port)
	log.Fatal(http.ListenAndServe(app.Port, mx))
}
