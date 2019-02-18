package components

// TODO: lists
// TODO: javascript components (ui)
// TODO: less in components > build
// TODO: save data
import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
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
	c.TemplateManager = templates.TemplateManager{}
	c.TemplateManager.Preload(path)
	c.TemplateManager.LocalizationData = make([]map[string]interface{}, 0)
	c.TemplateManager.Debug = true //TODO make false
	return c, nil
}

//Component struct
type Component struct {
	Path            string
	GetSQL          string
	TemplateManager templates.TemplateManager
}

//Name returns name from path
func (c *Component) Name() string {
	spl := strings.Split(c.Path, "/")
	return strings.ToLower(spl[len(spl)-1])
}

//GetData gets data. keys from url path
func (c *Component) GetData(path string) (map[string]interface{}, error) {
	var ret = make(map[string]interface{})
	if c.GetSQL == "" {
		return ret, nil
	}
	spl := strings.Split(path, "/")
	keys := strings.Split(spl[len(spl)-1], ":")
	params := make([]interface{}, len(keys))
	for i := range keys {
		params[i] = dbmodel.Escape(keys[i])
	}
	query := fmt.Sprintf(c.GetSQL, params...)
	// log.Println("DEBUG:", query)
	res, err := dbmodel.DoQuery(query)
	if err != nil {
		return ret, err
	}
	if len(res) == 0 {
		return ret, errors.New("Data not found")
	}
	return res[0], nil
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

		log.Println("DEBUG:", templateName)
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
		tmpl.Data = data
		html, err := c.TemplateManager.Render(&tmpl, "nl")
		if err != nil {
			log.Println("Error:", err)
			http.NotFound(w, r)
			return
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
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Write(bytes)
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
}
