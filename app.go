package components

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jmu0/dbAPI/db"

	"git.muysers.nl/jmu0/jwt"
	"github.com/jmu0/settings"
	"github.com/jmu0/templates"
	"github.com/tdewolff/minify"
	"github.com/tdewolff/minify/js"
)

//DataFunc function for getting data for component
type DataFunc func(keys []string, conn db.Conn) ([]map[string]interface{}, error)

var templateCache []byte

//App struct for app data
type App struct {
	Title           string   `json:"title" yaml:"title"`
	ComponentsPath  string   `json:"components_path" yaml:"components_path"` //use this or componentpaths
	ComponentPaths  []string `json:"componentpaths" yaml:"componentpaths"`   //use this or componentspath
	StaticPath      string   `json:"static_path" yaml:"static_path"`
	MainPath        string   `json:"main" yaml:"main"`
	Scripts         []string `json:"scripts" yaml:"scripts"`
	Debug           bool     `json:"debug" yaml:"debug"`
	ConfigFile      string
	Mux             *http.ServeMux
	Components      map[string]Component
	Pages           []Page
	TemplateManager templates.TemplateManager
	JsCache         []byte
	Port            string `json:"port" yaml:"port"`
	StartTime       time.Time
	RootPath        string
	Conn            db.Conn
	DataFuncs       map[string]DataFunc
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
	if a.TemplateManager.Cache == nil {
		a.TemplateManager.Cache = make(map[string]*templates.Template)
	}
	var main = templates.Template{}
	main.Data = make(map[string]interface{})
	err = main.Load(a.RootPath + a.ComponentsPath + "/" + a.MainPath)
	if err != nil {
		return err
	}
	main.Data["scripts"] = a.ScriptTags() //strings.Join(a.ScriptTags(), "\n")
	main.Data["templates"] = a.TemplateTags()
	main.Data["title"] = a.Title
	if a.Debug == true {
		main.Data["debug"] = "true"
	} else {
		main.Data["debug"] = "false"
	}
	if len(a.TemplateManager.LocalizationData) > 0 {
		bytes, err := json.Marshal(a.TemplateManager.LocalizationData)
		if err == nil {
			main.Data["localizationJSON"] = string(bytes)
		}
	}
	a.TemplateManager.Cache["main"] = &main
	a.StartTime = time.Now()
	return nil
}

//LoadConfig loads json config file
func (a *App) LoadConfig() error {
	settings.Load(a.RootPath+a.ConfigFile, a)
	if a.Debug == true {
		a.Scripts = append(a.Scripts, "/static/js/reload.socket.js")
	}
	return nil
}

//LoadComponents loads components from path
func (a *App) LoadComponents() error {
	a.Components = make(map[string]Component)
	var paths []string
	if a.ComponentPaths != nil {
		paths = a.ComponentPaths
	} else {
		paths = append(paths, a.ComponentsPath)
	}
	for _, path := range paths {
		files, err := ioutil.ReadDir(a.RootPath + path)
		if err != nil {
			return err
		}
		for _, file := range files {
			if file.IsDir() {
				err = a.loadComponentFolder(a.RootPath + path + "/" + file.Name())
				if err != nil {
					return err
				}
			}
		}
		if path != "components" { //or dirs in /components get loaded twice
			a.loadComponentFolder(a.RootPath + path)
		}
	}
	return nil
}

//loadComponentFolder recursive function to load components
func (a *App) loadComponentFolder(path string) error {
	c, err := a.loadComponent(path)
	if err != nil {
		return err
	}
	if !(len(c.JsFiles) == 0 && len(c.LessFiles) == 0 && len(c.TemplateManager.GetTemplates()) == 0) { //is a component
		if a.ComponentsPath != "" {
			c.Name = strings.Replace(path, a.RootPath+a.ComponentsPath, "", 1)
		} else {
			c.Name = strings.Replace(path, a.RootPath, "", 1)
			for _, cmppath := range a.ComponentPaths {
				if strings.Index(c.Name, cmppath) == 0 {
					if c.Name == cmppath {
						spl := strings.Split(cmppath, "components")
						if len(spl) > 0 {
							c.Name = spl[1]
						}
					} else {
						c.Name = strings.Replace(c.Name, cmppath, "", 1)
					}
					break
				}
			}
		}
		if len(c.Name) > 0 { //can be "" if path == components
			if c.Name[:1] == "/" {
				c.Name = c.Name[1:]
			}
			c.Name = strings.Replace(c.Name, "/", ".", -1)
			if f, ok := a.DataFuncs[c.Name]; ok {
				c.DataFunc = f
			}
			a.Components[c.Name] = c
			log.Println("Loading component:", c.Name, "from", c.Path)
		}
	}

	//scan directories in component folder
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return err
	}
	for _, file := range files {
		if file.IsDir() {
			log.Println("DEBUG scanning:", file.Name(), "in", path)
			err = a.loadComponentFolder(path + "/" + file.Name())
			if err != nil {
				return err
			}
		}
	}
	return nil
}

//LoadComponent loads component from files in <path>
func (a *App) loadComponent(path string) (Component, error) {
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
	c.TemplateManager.LocalizationData = a.TemplateManager.LocalizationData
	return c, nil
}

func (a *App) handleFunc(page Page) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var locale = "nl"
		val, err := jwt.GetPayloadValue(r, "locale")
		if err == nil {
			locale = val
		} else {
			if loc, ok := r.URL.Query()["locale"]; ok {
				locale = strings.Join(loc, "")
			}
		}
		if page.Auth == true {
			if jwt.Authenticated(r) == false {
				//render login component, if it exists
				if login, ok := a.Components["login"]; ok {
					data := make(map[string]interface{})
					data["error"] = r.Header.Get("error")
					data["uri"] = r.URL.Path
					log.Println("Rendering login")
					html, err := login.Render("", locale, data)
					if err == nil {
						w.Write([]byte(html))
						return
					}
				}
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}
		log.Println("Rendering", page.Route)
		content, err := page.Render(r.URL.Path, locale, a.Components, a.Conn)
		if err != nil {
			log.Println("ERROR:", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		a.TemplateManager.Cache["main"].Data["content"] = content
		html, err := a.TemplateManager.Render(a.TemplateManager.Cache["main"], locale)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(html))
	}
}

//AddRoutes adds routes for app
func (a *App) AddRoutes(conn db.Conn) error {
	//Add route for static path
	if a.StaticPath != "" {
		log.Println("Adding route for: favicon.ico")
		a.Mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-control", "max-age=86400")
			http.FileServer(http.Dir(a.RootPath+a.StaticPath)).ServeHTTP(w, r)
		})
		log.Println("Adding route for:", a.StaticPath)
		a.Mux.HandleFunc("/"+a.StaticPath+"/", func(w http.ResponseWriter, r *http.Request) {
			log.Println("Serving:", r.URL.Path)
			w.Header().Set("Cache-control", "max-age=90")
			if a.RootPath == "" {
				http.FileServer(http.Dir("./")).ServeHTTP(w, r)
			} else {
				http.FileServer(http.Dir(a.RootPath)).ServeHTTP(w, r)
			}
		})
	}

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
		if len(page.Route) > 1 {
			deRoute := a.TemplateManager.Translate(strings.Replace(page.Route, "/", "", -1), "de")
			if deRoute != strings.Replace(page.Route, "/", "", -1) {
				deRoute = "/" + strings.Replace(deRoute, " ", "", -1) + "/"
				log.Println("Adding route for page:", deRoute)
				a.Mux.HandleFunc(deRoute, a.handleFunc(page))
			}
			enRoute := a.TemplateManager.Translate(strings.Replace(page.Route, "/", "", -1), "en")
			if enRoute != strings.Replace(page.Route, "/", "", -1) {
				enRoute = "/" + strings.Replace(enRoute, " ", "", -1) + "/"
				log.Println("Adding route for page:", enRoute)
				a.Mux.HandleFunc(enRoute, a.handleFunc(page))
			}
		}
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
		if len(templateCache) == 0 {
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
			bytes, err = Compress(bytes)
			if err != nil {
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			log.Println("Serving: /component/templates: Compressed templates")
			templateCache = bytes
		} else {
			log.Println("Serving: /component/templates from cache")
		}
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Cache-control", "max-age=90")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Write(templateCache)
	})

	//Add API routes
	AddAPIRoutes(a.Mux, a.Conn)

	return nil
}

//ScriptTags returns html script tags for javascript files
func (a *App) ScriptTags() string {
	var ret, src string
	var i int
	var html string
	if a.Debug == true {
		for _, scriptPath := range a.Scripts {
			ret += "<script src=\"" + scriptPath + "\"></script>\n"
		}
		for _, cmp := range a.Components {
			for i = 0; i < len(cmp.JsFiles); i++ {
				src = strings.Replace(cmp.JsFiles[i], a.RootPath, "", -1)
				html = "<script src=\"/" + src + "\""
				if strings.Contains(src, "index") == false {
					html += "\" type=\"module\""
				}
				html += "></script>\n"
				ret += html
			}
		}
	} else {
		ret = "<script src=\"/static/js/" + a.Title + ".js\"></script>\n"
	}
	return ret
}

//TemplateTags returns <template> tags for each component
func (a *App) TemplateTags() string {
	var ret string
	var id string
	for _, comp := range a.Components {
		split := strings.Split(comp.Name, ".")
		for tmplname := range comp.TemplateManager.GetTemplates() {
			tmpl, err := comp.TemplateManager.GetTemplate(tmplname)
			if err == nil {
				if len(split) > 1 {
					id = strings.Join(split[:len(split)-1], ".") + "." + tmplname
				} else {
					id = tmplname
				}
				ret += "<template id=\"" + id + "\">" + tmpl.HTML + "</template>\n"
			}
		}
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
	comp, err := Compress(a.JsCache)
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

//Compress compresses a []byte
func Compress(inp []byte) ([]byte, error) {
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
