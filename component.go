package components

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/jmu0/dbAPI/db"
	"github.com/jmu0/templates"
)

//Component struct
type Component struct {
	Path            string
	Name            string
	TemplateManager templates.TemplateManager
	LessFiles       []string
	JsFiles         []string
	DataFunc        DataFunc
}

//OldName returns name from path
func (c *Component) OldName() string {
	spl := strings.Split(c.Path, "/")
	return strings.ToLower(spl[len(spl)-1])
}

//GetData gets data. keys from url path
func (c *Component) GetData(path string, conn db.Conn) ([]map[string]interface{}, error) {
	var ret = make([]map[string]interface{}, 0)
	if c.DataFunc == nil {
		return ret, errors.New("No DataFunc for component: " + c.Name)
	}
	var param string
	spl := strings.Split(path, "/")
	keys := strings.Split(spl[len(spl)-1], ":")
	params := make([]string, 0)
	for i := range keys {
		param = db.Escape(strings.TrimSpace(keys[i]))
		if len(param) > 0 {
			params = append(params, param)
		}
	}
	return c.DataFunc(params, conn)
}

//Render renders the component
func (c *Component) Render(templateName, locale string, data map[string]interface{}) (string, error) {
	tmpl, err := c.TemplateManager.GetTemplate(templateName)
	if err != nil {
		//get first template in cache if not found
		for _, first := range c.TemplateManager.Cache {
			first.Data = data
			return c.TemplateManager.Render(first, locale)
		}
		return "", err
	}
	tmpl.Data = data
	return c.TemplateManager.Render(tmpl, locale)
}

//Render renders component (prevent closure in loop over templates)
func handleFunc(c Component, templateName string, conn db.Conn) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var html, itemhtml string
		spl := strings.Split(r.URL.Path, "/")
		if spl[len(spl)-1] == templateName {
			log.Println("Error: no key for " + templateName)
			http.NotFound(w, r)
			return
		}

		data, err := c.GetData(r.URL.Path, conn)
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
			html, err = c.TemplateManager.Render(tmpl, "nl")
			if err != nil {
				log.Println("Error:", err)
				http.NotFound(w, r)
				return
			}
		} else if len(data) > 1 {
			for i := range data {
				tmpl.Data = data[i]
				itemhtml, err = c.TemplateManager.Render(tmpl, "nl")
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

func handleFuncData(c Component, conn db.Conn) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := c.GetData(r.URL.Path, conn)
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
func (c *Component) AddRoutesComponent(mx *http.ServeMux, conn db.Conn) {
	var route string
	split := strings.Split(c.Name, ".")
	for name := range c.TemplateManager.GetTemplates() {
		if len(split) > 1 {
			route = strings.Join(split[:len(split)-1], "/") + "/" + name
		} else {
			route = name
		}
		log.Println("Adding route for component: /component/" + route + "/")
		mx.HandleFunc("/component/"+route+"/", handleFunc(*c, name, conn))
		if len(split) > 1 {
			route = strings.Join(split[:len(split)-1], ".") + "." + name
		} else {
			route = name
		}
		log.Println("Adding route for template: /static/templates/" + route + ".html")
		mx.HandleFunc("/static/templates/"+route+".html", handleFuncTemplate(*c, name))
	}
}

//AddRoutesScripts adds Routes for js files
func (c *Component) AddRoutesScripts(mx *http.ServeMux, rootPath string) {
	if len(c.JsFiles) > 0 {
		var route string
		var i int
		for i = 0; i < len(c.JsFiles); i++ {
			route = "/" + strings.Replace(c.JsFiles[i], rootPath, "", -1)
			log.Println("Adding route for script:" + route)
			mx.HandleFunc(route, handleFuncScript(c.JsFiles[i]))
		}
	}
}
