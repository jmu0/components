package components

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/jmu0/orm/dbmodel"
	"github.com/jmu0/templates"
)

//LoadComponent loads component from files in <path>
func LoadComponent(path string) (Component, error) {
	var c = Component{
		Path: path,
	}
	if _, err := os.Stat(path + "/api.yml"); err == nil {
		err = LoadRoutesYaml(path + "/api.yml")
		if err != nil {
			return c, err
		}
	}

	lessfiles, err := filepath.Glob(c.Path + "/*.less")
	if len(lessfiles) > 0 && err == nil {
		c.LessFiles = lessfiles
	}
	jsfiles, err := filepath.Glob(c.Path + "/*.js")
	if len(jsfiles) > 0 && err == nil {
		c.JsFiles = jsfiles
	}
	c.TemplateManager = templates.TemplateManager{}
	c.TemplateManager.Preload(path)
	c.TemplateManager.LocalizationData = make([]map[string]interface{}, 0) //TODO: localization
	return c, nil
}

//Component struct
type Component struct {
	Path            string
	Name            string
	TemplateManager templates.TemplateManager
	LessFiles       []string
	JsFiles         []string
}

//OldName returns name from path
func (c *Component) OldName() string {
	spl := strings.Split(c.Path, "/")
	return strings.ToLower(spl[len(spl)-1])
}

//GetData gets data. keys from url path
func (c *Component) GetData(path string) ([]map[string]interface{}, error) {
	var ret = make([]map[string]interface{}, 0)
	var query, param string
	if route, ok := routes[c.Name]; ok {
		if route.Type == "query" && len(route.SQL) > 0 {
			query = route.SQL
		} else {
			return ret, errors.New("No query found in route: " + route.Route)
		}
	} else {
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
	if len(params) > 0 {
		query = fmt.Sprintf(query, params...)
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

//Render renders the component
func (c *Component) Render(templateName string, data map[string]interface{}) (string, error) {
	tmpl, err := c.TemplateManager.GetTemplate(templateName)
	if err != nil {
		return "", err
	}
	tmpl.Data = data
	return c.TemplateManager.Render(&tmpl, "nl")
}

//Render renders component (prevent closure in loop over templates)
func handleFunc(c Component, templateName string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var html, itemhtml string
		spl := strings.Split(r.URL.Path, "/")
		if spl[len(spl)-1] == templateName {
			log.Println("Error: no key for " + templateName)
			http.NotFound(w, r)
			return
		}
		data, err := c.GetData(r.URL.Path)
		if err != nil {

			log.Println("Query error:", err)
			http.NotFound(w, r)
			return
		}
		tmpl, err := c.TemplateManager.GetTemplate(templateName)
		if err != nil {
			log.Println("Error:", err)
			http.NotFound(w, r)
			return
		}
		if len(data) <= 1 {
			if len(data) == 1 {
				tmpl.Data = data[0]
			}
			html, err = c.TemplateManager.Render(&tmpl, "nl")
			if err != nil {
				log.Println("Error:", err)
				http.NotFound(w, r)
				return
			}
		} else if len(data) > 1 {
			for i := range data {
				tmpl.Data = data[i]
				itemhtml, err = c.TemplateManager.Render(&tmpl, "nl")
				if err != nil {
					log.Println("Error:", err)
					http.NotFound(w, r)
					return
				}
				html += itemhtml
			}
		}
		log.Println("Serving ", r.URL.Path)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(html))
	}
}

func handleFuncData(c Component) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := c.GetData(r.URL.Path)
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

func handleFuncScript(s string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, s)
	}
}

func handleFuncTemplate(c Component, name string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := c.TemplateManager.GetTemplate(name)
		if err != nil {
			log.Println("ERROR:", err)
			http.NotFound(w, r)
			return
		}
		log.Println("Serving template:", name)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(tmpl.HTML))
	}
}

//AddRoutesComponent adds routes for html endpoints
func (c *Component) AddRoutesComponent(mx *http.ServeMux) {
	var route string
	split := strings.Split(c.Name, ".")
	for name := range c.TemplateManager.GetTemplates() {
		if len(split) > 1 {
			route = strings.Join(split[:len(split)-1], "/") + "/" + name
		} else {
			route = name
		}
		log.Println("Adding route /component/" + route + "/")
		mx.HandleFunc("/component/"+route+"/", handleFunc(*c, name))
		if len(split) > 1 {
			route = strings.Join(split[:len(split)-1], ".") + "." + name
		} else {
			route = name
		}
		log.Println("Adding route /static/templates/" + route + ".html")
		mx.HandleFunc("/static/templates/"+route+".html", handleFuncTemplate(*c, name))
	}
}

//AddRoutesScripts adds Routes for js files
func (c *Component) AddRoutesScripts(mx *http.ServeMux) {
	if len(c.JsFiles) > 0 {
		var route string
		var i int
		for i = 0; i < len(c.JsFiles); i++ {

			/* route in /static/js path
			split := strings.Split(c.Name, ".")
			route = "/static/js/"
			if len(split) > 1 {
				route += strings.Join(split[:len(split)-1], "/") + "/"
			}
			route += filepath.Base(c.JsFiles[i])
			//*/

			//* route in original components path
			route = "/" + c.JsFiles[i]
			//*/

			log.Println("Adding route " + route)
			mx.HandleFunc(route, handleFuncScript(c.JsFiles[i]))
		}
	}
}
