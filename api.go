package components

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"git.muysers.nl/jmu0/jwt"

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

var routes map[string]Route
var apiURL = "/api"

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
	if routes == nil {
		routes = make(map[string]Route)
	}
	for _, rt := range rts {
		routes[rt.Route] = rt
	}
	return nil
}

//AddAPIRoutes creates handlers for routes
func AddAPIRoutes(mx *http.ServeMux) {
	for _, r := range routes {
		switch r.Type {
		case "query":
			log.Println("Adding route for api: /api/" + r.Route + "/ (" + r.Type + ")")
			mx.HandleFunc("/api/"+r.Route+"/", queryHandler(r))
		case "rest":
			log.Println("Adding route for api: /api/" + r.Route + "/ (" + r.Type + ")")
			mx.HandleFunc("/api/"+r.Route+"/", restHandler(r))
		default:
			log.Println("ERROR unknown route type:", r.Type)
		}
	}
}

//restHandler handler for rest api requests
func restHandler(route Route) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var allow = false
		if strings.Contains(strings.ToLower(route.Methods), strings.ToLower(r.Method)) {
			allow = true
		}
		if route.Auth == true {
			if jwt.Authenticated(r) == false {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}
		if allow == true {
			dbmodel.HandleREST(apiURL, w, r)
		} else {
			log.Println("Method not allowed:", r.Method, r.URL.Path)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			// http.NotFound(w, r)
		}
	}
}

//queryHandler creates handler func for query route
func queryHandler(route Route) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if route.Auth == true {
			if jwt.Authenticated(r) == false {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}
		data, err := route.GetData(r.URL.Path)
		if err != nil {
			log.Println("Error handle data:", err)
			http.NotFound(w, r)
			return
		}
		bytes, err := json.Marshal(data)
		if err != nil {
			log.Println("Error building json:", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		log.Println("Serving data:", r.URL.Path)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Write(bytes)
	}
}

//GetData gets data. keys from url path
func (r *Route) GetData(path string) ([]map[string]interface{}, error) {
	var ret = make([]map[string]interface{}, 0)
	var query, param string
	if r.SQL == "" {
		return ret, nil
	}
	spl := strings.Split(path, "/")
	keys := strings.Split(spl[len(spl)-1], ":")
	params := make([]interface{}, 0)
	for i := range keys {
		param = dbmodel.Escape(strings.TrimSpace(keys[i]))
		if len(param) > 0 {
			params = append(params, param)
		}
	}
	if len(params) == 0 {
		query = r.SQL
	} else {
		query = fmt.Sprintf(r.SQL, params...)
	}
	res, err := dbmodel.DoQuery(query)
	if err != nil {
		return ret, err
	}
	if len(res) == 0 {
		return ret, errors.New("Data not found")
	}
	return res, nil
}
