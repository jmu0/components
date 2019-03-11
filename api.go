package components

import (
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/jmu0/orm/dbmodel"
	"gopkg.in/yaml.v2"
)

//Route struct for api route data
type Route struct {
	Route   string `yaml:"route"`
	Type    string `yaml:"type"`
	Auth    bool   `yaml:"auth"`
	Methods string `yaml:"methods"`
	SQL     string `yaml:"sql"`
}

var routes []Route

//LoadRoutesYaml loads routes from yaml file and adds to routes slice
func LoadRoutesYaml(path string) error {
	// log.Println("Loading routes from", path)
	yml, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	var rts []Route
	err = yaml.Unmarshal(yml, &rts)
	if err != nil {
		return err
	}
	routes = append(routes, rts...)
	return nil
}

//AddAPIRoutes creates handlers for routes
func AddAPIRoutes(mx *http.ServeMux) {
	for _, r := range routes {
		log.Println("TODO: create route:", r)
	}
}

//RestHandler handler for rest api requests
func RestHandler() func(w http.ResponseWriter, r *http.Request) {
	var dataURL string
	var allowed = make(map[string]string)
	return func(w http.ResponseWriter, r *http.Request) {
		var allow = false
		var reqPath = strings.Replace(r.URL.Path, dataURL, "", 1)
		var method = r.Method
		for path, methods := range allowed {
			if len(path) < len(reqPath) && path == reqPath[:len(path)] {
				if strings.Contains(strings.ToLower(methods), strings.ToLower(method)) {
					allow = true
					break
				}
			}
		}
		if allow == true {
			dbmodel.HandleREST(dataURL, w, r)
		} else {
			log.Println("NOT ALLOWED:", method, reqPath)
			http.NotFound(w, r)
		}
	}
}
