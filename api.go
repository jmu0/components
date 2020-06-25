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
	"github.com/graphql-go/graphql"
	"github.com/jmu0/dbAPI/api"
	"github.com/jmu0/dbAPI/db"

	// "github.com/graphql-go/graphql"

	"gopkg.in/yaml.v2"
)

//Route struct for api route data
type Route struct {
	Route   string   `yaml:"route"`
	Type    string   `yaml:"type"`
	Auth    bool     `yaml:"auth"`
	Methods string   `yaml:"methods"`
	SQL     string   `yaml:"sql"`
	Tables  []string `yaml:"tables"`
}

var routes map[string]*Route
var apiURL = "/api"

//LoadRoutesYaml loads routes from yaml file and adds to routes slice
func LoadRoutesYaml(path string) error {
	// log.Println("Loading routes from", path)
	yml, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	var rts []*Route
	err = yaml.Unmarshal(yml, &rts)
	if err != nil {
		return err
	}
	if routes == nil {
		routes = make(map[string]*Route)
	}
	for _, rt := range rts {
		if rt.Type == "graphql" {
			if _, ok := routes[rt.Route]; ok {
				routes[rt.Route].Tables = append(routes[rt.Route].Tables, rt.Tables...)
				if routes[rt.Route].Auth == false && rt.Auth == true {
					routes[rt.Route].Auth = true
				}
				// log.Println("DEBUG added tables to route", rt.Route, rt.Tables)
			} else {
				routes[rt.Route] = rt
				// log.Println("DEBUG created route:", rt.Route)
			}
		} else {
			routes[rt.Route] = rt
		}
	}
	return nil
}

//AddAPIRoutes creates handlers for routes
func AddAPIRoutes(mx *http.ServeMux, conn db.Conn) {
	// log.Println("DEBUG Routes", routes)
	// log.Println("DEBUG graphql tables", routes["graphql"].Tables)
	for _, r := range routes {
		// log.Println("DEBUG r=", r)
		switch r.Type {
		case "query":
			log.Println("Adding route for api: /api/"+r.Route+"/ ("+r.Type+")", "auth:", r.Auth)
			mx.HandleFunc("/api/"+r.Route+"/", queryHandler(*r, conn))
		case "rest":
			log.Println("Adding route for api: /api/"+r.Route+"/ ("+r.Type+")", "auth:", r.Auth)
			mx.HandleFunc("/api/"+r.Route+"/", restHandler(*r, conn))
		case "graphql":
			log.Println("Adding route for api: /api/"+r.Route+" ("+r.Type+")", "auth:", r.Auth)
			schema, err := api.BuildSchema(api.BuildSchemaArgs{
				Tables: r.Tables,
				Conn:   conn,
			})
			if err != nil {
				log.Println("GraphQL Schema error:", err)
			}
			mx.HandleFunc("/api/"+r.Route, graphQLhandler(*r, &schema))
		default:
			log.Println("ERROR unknown route type:", r.Type)
		}

	}
}

//restHandler handler for rest api requests
func restHandler(route Route, conn db.Conn) func(w http.ResponseWriter, r *http.Request) {
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
			api.RestHandler(apiURL, conn)(w, r)
			// api.HandleREST(apiURL, w, r)
		} else {
			log.Println("Method not allowed:", r.Method, r.URL.Path)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			// http.NotFound(w, r)
		}
	}
}

//queryHandler creates handler func for query route
func queryHandler(route Route, conn db.Conn) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if route.Auth == true {
			if jwt.Authenticated(r) == false {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}

		data, err := route.GetData(r.URL.Path, conn)
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

func graphQLhandler(route Route, schema *graphql.Schema) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if route.Auth == true {
			if jwt.Authenticated(r) == false {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}
		api.HandleGQL(schema, w, r)
	}
}

//GetData gets data. keys from url path
func (r *Route) GetData(path string, conn db.Conn) ([]map[string]interface{}, error) {
	var ret = make([]map[string]interface{}, 0)
	var query, param string
	params := make([]interface{}, 0)
	if r.SQL == "" {
		return ret, nil
	}
	if path[len(path)-1:] != "/" {
		spl := strings.Split(path, "/")
		keys := strings.Split(spl[len(spl)-1], ":")
		for i := range keys {
			param = db.Escape(strings.TrimSpace(keys[i]))
			if len(param) > 0 {
				params = append(params, param)
			}
		}
	}
	if len(params) == 0 {
		query = r.SQL
	} else {
		query = fmt.Sprintf(r.SQL, params...)
	}
	if strings.ToLower(strings.TrimSpace(query)[:6]) == "select" {
		res, err := conn.Query(query)
		if err != nil {
			return ret, err
		}
		if len(res) == 0 {
			return ret, errors.New("Data not found: " + path)
		}
		ret = res
	} else {
		id, rows, err := conn.Execute(query)
		if err != nil {
			return ret, err
		}
		ret = append(ret, make(map[string]interface{}))
		ret[0]["id"] = id
		ret[0]["n"] = rows
	}
	return ret, nil
}
