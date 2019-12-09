package components

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/jmu0/dbAPI/db"

	"git.muysers.nl/jmu0/jwt"
	"github.com/jmu0/settings"
	"github.com/jmu0/templates"
	"github.com/tdewolff/minify"
	"github.com/tdewolff/minify/js"
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
	Port           string `json:"port" yaml:"port"`
	StartTime      time.Time
	RootPath       string
	Conn           db.Conn
}

//Init initializes the app
func (a *App) Init() error {
	err := a.LoadConfig()
	if err != nil {
		return err
	}
	if a.Port == "" {
		a.Port = ":8080"
	}
	err = a.LoadComponents()
	if err != nil {
		return err
	}
	err = a.AddRoutes(a.Conn)
	if err != nil {
		return err
	}
	var main = templates.Template{}
	main.Data = make(map[string]interface{})
	err = main.Load(a.RootPath + a.ComponentsPath + "/" + a.MainPath)
	if err != nil {
		return err
	}
	main.Data["scripts"] = a.ScriptTags() //strings.Join(a.ScriptTags(), "\n")
	main.Data["title"] = a.Title
	a.MainTemplate = &main
	a.StartTime = time.Now()
	return nil
}

//LoadConfig loads json config file
func (a *App) LoadConfig() error {
	settings.Load(a.RootPath+a.ConfigFile, a)
	// if path.Ext(a.RootPath+a.ConfigFile) == ".json" {
	// 	log.Println("Loading config file:", a.RootPath+a.ConfigFile)
	// 	bytes, err := ioutil.ReadFile(a.RootPath + a.ConfigFile)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	err = json.Unmarshal(bytes, a)
	// 	if err != nil {
	// 		return err
	// 	}
	// } else if path.Ext(a.RootPath+a.ConfigFile) == ".yml" {
	// 	log.Println("Loading config file:", a.RootPath+a.ConfigFile)
	// 	yml, err := ioutil.ReadFile(a.RootPath + a.ConfigFile)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	err = yaml.Unmarshal(yml, a)
	// 	if err != nil {
	// 		return err
	// 	}
	// } else {
	// 	return errors.New("Invalid config file: " + a.RootPath + a.ConfigFile)
	// }
	if a.Debug == true {
		a.Scripts = append(a.Scripts, a.RootPath+"/static/js/reload.socket.js")
	}
	return nil
}

//LoadComponents loads components from path
func (a *App) LoadComponents() error {
	a.Components = make(map[string]Component)
	files, err := ioutil.ReadDir(a.RootPath + a.ComponentsPath)
	if err != nil {
		return err
	}
	for _, file := range files {
		if file.IsDir() {
			err = a.loadComponentFolder(a.RootPath + a.ComponentsPath + "/" + file.Name())
			if err != nil {
				return err
			}
		}
	}
	return nil
}

//loadComponentFolder recursive function to load components
func (a *App) loadComponentFolder(path string) error {
	c, err := LoadComponent(path)
	if err != nil {
		return err
	}
	if !(len(c.JsFiles) == 0 && len(c.LessFiles) == 0 && len(c.TemplateManager.GetTemplates()) == 0) { //not a template
		c.Name = strings.Replace(path, a.RootPath+a.ComponentsPath, "", 1)
		if c.Name[:1] == "/" {
			c.Name = c.Name[1:]
		}
		c.Name = strings.Replace(c.Name, "/", ".", -1)
		a.Components[c.Name] = c
		log.Println("Loading component:", c.Name, "from", c.Path)
	} else {
		log.Println("DEBUG: not a component:", c.Path)
	}

	//scan directories in component folder
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return err
	}
	for _, file := range files {
		if file.IsDir() {
			err = a.loadComponentFolder(path + "/" + file.Name())
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (a *App) handleFunc(page Page) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if page.Auth == true {
			if jwt.Authenticated(r) == false {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}
		log.Println("Rendering", page.Route)
		content, err := page.Render(a.Components, r.URL.Path, a.Conn)
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
func (a *App) AddRoutes(conn db.Conn) error {
	//Add routes for Pages
	for _, page := range a.Pages {
		if len(page.Route) == 0 {
			return errors.New("No route given for page, check config")
		}
		if page.Route[len(page.Route)-1] != '/' {
			page.Route += "/"
		}
		log.Println("Adding route for page:", page.Route)
		a.Mux.HandleFunc(page.Route, a.handleFunc(page))
	}
	//Add routes for components, data and scripts
	for _, comp := range a.Components {
		comp.AddRoutesComponent(a.Mux, conn)
		if a.Debug == true {
			comp.AddRoutesScripts(a.Mux, a.RootPath)
		}
	}
	if a.Debug == false {
		a.LoadScriptCache()
		log.Println("Adding route for script: /static/js/" + a.Title + ".js")
		a.Mux.HandleFunc("/static/js/"+a.Title+".js", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-control", "max-age=90")
			w.Header().Set("Last-Modified", a.StartTime.UTC().Format(http.TimeFormat))
			w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
			w.Header().Set("Content-Encoding", "gzip")
			w.Write(a.JsCache)
		})
	} else {
		//serve reload socket script
		log.Println("Adding route for reload socket script: /static/js/reload.socket.js")
		a.Mux.HandleFunc("/static/js/reload.socket.js", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
			w.Write(reloadSocketScript())
		})
	}

	//Add route for templates
	log.Println("Adding route for template collection: /component/templates")
	a.Mux.HandleFunc("/component/templates", func(w http.ResponseWriter, r *http.Request) {
		//TODO template cache?
		tmpls := make(map[string]string)
		for _, comp := range a.Components {
			split := strings.Split(comp.Name, ".")
			for tmplname := range comp.TemplateManager.GetTemplates() {
				tmpl, err := comp.TemplateManager.GetTemplate(tmplname)
				if err == nil {
					if len(split) > 1 {
						tmpls[strings.Join(split[:len(split)-1], ".")+"."+tmplname] = tmpl.HTML
					} else {
						tmpls[tmplname] = tmpl.HTML
					}
				}
			}
		}
		bytes, err := json.Marshal(tmpls)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		zip, err := compress(bytes)
		if err == nil {
			bytes = zip
			log.Println("Compressed templates")
			w.Header().Set("Content-Encoding", "gzip")
		} else {
			log.Println("Error compressing templates:", err)
		}
		w.Header().Set("Cache-control", "max-age=90")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Write(bytes)
	})

	//Add API routes
	AddAPIRoutes(a.Mux, a.Conn)

	return nil
}

//ScriptTags returns html script tags for javascript files
func (a *App) ScriptTags() string {
	var ret string
	var i int
	var html string
	if a.Debug == true {
		for _, scriptPath := range a.Scripts {
			ret += "<script src=\"" + scriptPath + "\"></script>\n"
		}
		for _, cmp := range a.Components {
			for i = 0; i < len(cmp.JsFiles); i++ {
				html = "<script src=\"/" + cmp.JsFiles[i]
				html += "\"></script>\n"
				ret += html
			}
		}
	} else {
		ret = "<script src=\"/static/js/" + a.Title + ".js\"></script>\n"
	}
	return ret
}

//LoadScriptCache loads and crushes js files
func (a *App) LoadScriptCache() {
	a.JsCache = []byte("")
	for _, scriptPath := range a.Scripts {
		if scriptPath[0] == '/' {
			scriptPath = scriptPath[1:]
		}
		scriptPath = a.RootPath + scriptPath
		a.JsCache = append(a.JsCache, loadJsFile(scriptPath)...)
	}
	var i int
	for _, cmp := range a.Components {
		for i = 0; i < len(cmp.JsFiles); i++ {
			a.JsCache = append(a.JsCache, loadJsFile(cmp.JsFiles[i])...)
		}
	}
	comp, err := compress(a.JsCache)
	if err == nil {
		a.JsCache = comp
		log.Println("Compressed script cache")
	} else {
		log.Println("Error compressing script cache")
	}
}

func loadJsFile(path string) []byte {
	bytes, err := ioutil.ReadFile(path)
	log.Println("Loading script cache:", path)
	if err != nil {
		log.Println("Error loading script:", err)
	}
	m := minify.New()
	m.AddFunc("text/javascript", js.Minify)
	minified, err := m.String("text/javascript", string(bytes))
	if err != nil {
		minified = string(bytes)
		log.Println("ERROR minifying js file:", err)
	}
	return []byte(minified)
}

func compress(inp []byte) ([]byte, error) {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, err := zw.Write(inp)
	if err != nil {
		return inp, err
	}
	if err := zw.Close(); err != nil {
		return inp, err
	}
	return []byte(buf.String()), nil
}
