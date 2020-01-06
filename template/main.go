package main

import (
	"log"
	"net/http"

	"github.com/jmu0/components"
	"github.com/jmu0/dbAPI/api"
	"github.com/jmu0/dbAPI/db"
	"github.com/jmu0/settings"
)

var staticDir = "static"

//var listenAddr = ":8282"

var conn db.Conn
var err error
var mx *http.ServeMux

func main() {
	s := map[string]string{
		"root":   "./",
		"static": "static",
	}
	settings.Load("config.yml", &s)
	conn, err = api.GetConnection("config.yml")
	mx = http.NewServeMux()

	var app = components.App{
		ConfigFile: "app.yml",
		Mux:        mx,
		RootPath:   s["root"],
		StaticPath: s["static"],
		Conn:       conn,
		DataFuncs:  make(map[string]components.DataFunc),
	}
	app.DataFuncs["example"] = getExampleData

	err := app.Init()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Debug:", app.Debug)
	log.Println("Listening on port", app.Port)
	log.Fatal(http.ListenAndServe(app.Port, mx))
}

func getExampleData(keys []string, conn db.Conn) ([]map[string]interface{}, error) {
	log.Println("DEBUG getExampleData:", keys)
	var ret = make([]map[string]interface{}, 0)
	var one = make(map[string]interface{})
	if len(keys) > 0 {
		one["testkey"] = keys[0]
	}
	if len(keys) > 1 {
		one["namekey"] = keys[1]
	}
	one["idkey"] = "example from getExampleData func."
	ret = append(ret, one)
	return ret, nil
}
