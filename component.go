package components

// TODO: javascript components (ui)
// TODO: save data
// TODO: add example
// TODO: jwt auth
// TODO: build tool new (template)
import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
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
	if _, err := os.Stat(path + "/get.sql"); err == nil {
		bytes, err := ioutil.ReadFile(path + "/get.sql")
		if err != nil {
			return c, err
		}
		c.GetSQL = string(bytes)
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
	GetSQL          string
	TemplateManager templates.TemplateManager
	LessFiles       []string
	JsFiles         []string
}

//Name returns name from path
func (c *Component) Name() string {
	spl := strings.Split(c.Path, "/")
	return strings.ToLower(spl[len(spl)-1])
}

//GetData gets data. keys from url path
func (c *Component) GetData(path string) ([]map[string]interface{}, error) {
	var ret = make([]map[string]interface{}, 0)
	var query, param string
	if c.GetSQL == "" {
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
		query = c.GetSQL
	} else {
		query = fmt.Sprintf(c.GetSQL, params...)
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

//AddRoutes adds routes for html and json endpoints
func (c *Component) AddRoutes(mx *http.ServeMux) {
	for name := range c.TemplateManager.GetTemplates() {
		log.Println("Adding route /component/" + name + "/")
		mx.HandleFunc("/component/"+name+"/", handleFunc(*c, name)) //TODO inefficient
	}
	if c.GetSQL != "" {
		log.Println("Adding route /data/" + c.Name())
		mx.HandleFunc("/data/"+c.Name()+"/", handleFuncData(*c)) //TODO inefficient
	}
	if len(c.JsFiles) > 0 {
		var route string
		var i int
		for i = 0; i < len(c.JsFiles); i++ {
			route = "/static/js/"
			if filepath.Base(c.JsFiles[i]) == c.Name()+".js" {
				route = route + filepath.Base(c.JsFiles[i])
			} else {
				route = route + c.Name() + "." + filepath.Base(c.JsFiles[i])
			}
			log.Println("Adding route " + route)
			mx.HandleFunc(route, handleFuncScript(c.JsFiles[i]))
		}
	}
}
