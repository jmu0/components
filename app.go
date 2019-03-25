package components

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"path/filepath"
	"strings"

	"github.com/jmu0/templates"
	"github.com/tdewolff/minify"
	"github.com/tdewolff/minify/js"
	"gopkg.in/yaml.v2"
)

//App struct for app data
type App struct {
	Title          string   `json:"title" yaml:"title"`
	ComponentsPath string   `json:"components_path" yaml:"components_path"`
	MainPath       string   `json:"main" yaml:"main"`
	Scripts        []string `json:"scripts" yaml:"scripts"`
	Debug          bool     `json:"debug" yaml:"debug"`
	ConfigFile     string
	Mux            *http.ServeMux
	Components     map[string]Component
	Pages          []Page
	MainTemplate   *templates.Template
	JsCache        []byte
}

//Init initializes the app
func (a *App) Init() error {
	err := a.LoadConfig()
	if err != nil {
		return err
	}
	err = a.LoadComponents()
	if err != nil {
		return err
	}
	err = a.AddRoutes()
	if err != nil {
		return err
	}
	var main = templates.Template{}
	main.Data = make(map[string]interface{})
	err = main.Load(a.ComponentsPath + "/" + a.MainPath)
	if err != nil {
		return err
	}
	main.Data["scripts"] = strings.Join(a.ScriptTags(), "\n")
	main.Data["title"] = a.Title
	a.MainTemplate = &main
	// log.Println("DEBUG:", main.Data["scripts"])
	return nil
}

//LoadConfig loads json config file
func (a *App) LoadConfig() error {
	//TODO: load json or yaml format
	if path.Ext(a.ConfigFile) == ".json" {
		bytes, err := ioutil.ReadFile(a.ConfigFile)
		if err != nil {
			return err
		}
		err = json.Unmarshal(bytes, a)
		if err != nil {
			return err
		}
		return nil
	} else if path.Ext(a.ConfigFile) == ".yml" {
		yml, err := ioutil.ReadFile(a.ConfigFile)
		if err != nil {
			return err
		}
		err = yaml.Unmarshal(yml, a)
		if err != nil {
			return err
		}
		return nil
	}
	return errors.New("Invalid config file: " + a.ConfigFile)
}

//LoadComponents loads components from path
func (a *App) LoadComponents() error {
	a.Components = make(map[string]Component)
	files, err := ioutil.ReadDir(a.ComponentsPath)
	if err != nil {
		return err
	}
	//TODO: components in folders (recursive)
	for _, file := range files {
		if file.IsDir() {
			c, err := LoadComponent(a.ComponentsPath + "/" + file.Name())
			if err != nil {
				return err
			}
			a.Components[c.Name()] = c
		}
	}
	return nil
}

func (a *App) handleFunc(page Page) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if page.Auth == true {
			//TODO: jwt auth
			log.Println("TODO: Check auth..")
		}
		log.Println("Rendering", page.Route)
		content, err := page.Render(a.Components, r.URL.Path)
		if err != nil {
			log.Println("ERROR:", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		var tm = templates.TemplateManager{}
		a.MainTemplate.Data["content"] = content
		html, err := tm.Render(a.MainTemplate, "nl") //TODO: localize
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(html))
	}
}

//AddRoutes adds routes for app
func (a *App) AddRoutes() error {
	//Add routes for Pages
	for _, page := range a.Pages {
		if page.Route[len(page.Route)-1] != '/' {
			page.Route += "/"
		}
		log.Println("Adding route", page.Route)

		a.Mux.HandleFunc(page.Route, a.handleFunc(page))
	}
	//Add routes for components, data and scripts
	for _, comp := range a.Components {
		comp.AddRoutesData(a.Mux) //TODO to api
		if a.Debug == true {
			comp.AddRoutesScripts(a.Mux)
		}
	}
	if a.Debug == false {
		a.LoadScriptFiles()
		log.Println("Adding route /static/js/" + a.Title + ".js")
		a.Mux.HandleFunc("/static/js/"+a.Title+".js", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-control", "max-age=90")
			w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
			w.Write(a.JsCache)
		})
	}

	//Add route for templates
	log.Println("Adding route /component/templates")
	a.Mux.HandleFunc("/component/templates", func(w http.ResponseWriter, r *http.Request) {
		tmpls := make(map[string]string)
		for _, comp := range a.Components {
			for tmplname := range comp.TemplateManager.GetTemplates() {
				tmpl, err := comp.TemplateManager.GetTemplate(tmplname)
				if err == nil {
					tmpls[tmplname] = tmpl.HTML
				}
			}
		}
		bytes, err := json.Marshal(tmpls)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Cache-control", "max-age=90")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Write(bytes)
	})

	//Add API routes
	AddAPIRoutes(a.Mux)

	return nil
}

//ScriptTags returns html script tags for javascript files
func (a *App) ScriptTags() []string {
	var ret []string
	var i int
	var html string
	if a.Debug == true {
		for _, scriptPath := range a.Scripts {
			ret = append(ret, "<script src=\""+scriptPath+"\"></script>")
		}
		for _, cmp := range a.Components {
			// log.Println("DEBUG getting script tags for", name)
			for i = 0; i < len(cmp.JsFiles); i++ {
				html = "<script src=\"/static/js/"
				if filepath.Base(cmp.JsFiles[i]) == cmp.Name()+".js" {
					html += filepath.Base(cmp.JsFiles[i])
				} else {
					html += cmp.Name() + "." + filepath.Base(cmp.JsFiles[i])
				}
				html += "\"></script>"
				ret = append(ret, html)
			}
		}
	} else {
		ret = append(ret, "<script src=\"/static/js/"+a.Title+".js\"></script>")
	}
	return ret
}

//LoadScriptFiles loads and crushes js files
func (a *App) LoadScriptFiles() {

	a.JsCache = []byte("")
	for _, scriptPath := range a.Scripts {
		// log.Println("Loading Javascript:", scriptPath)
		if scriptPath[0] == '/' {
			scriptPath = scriptPath[1:]
		}
		a.JsCache = append(a.JsCache, loadJsFile(scriptPath)...)
	}
	var i int
	for _, cmp := range a.Components {
		for i = 0; i < len(cmp.JsFiles); i++ {
			// log.Println("Loading Javascript:", cmp.JsFiles[i])
			a.JsCache = append(a.JsCache, loadJsFile(cmp.JsFiles[i])...)
		}
	}
}

func loadJsFile(path string) []byte {
	bytes, err := ioutil.ReadFile(path)
	m := minify.New()
	m.AddFunc("text/javascript", js.Minify)
	minified, err := m.String("text/javascript", string(bytes))
	if err != nil {
		minified = string(bytes)
		log.Println("ERROR minifying js file:", err)
	}
	return []byte(minified)
}
