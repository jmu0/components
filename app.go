package components

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
)

//App struct for app data
type App struct {
	Title          string
	ComponentsPath string
	ConfigFile     string
	Mux            *http.ServeMux
	Components     map[string]Component
	Pages          []Page
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
	return nil
}

//LoadConfig loads json config file
func (a *App) LoadConfig() error {
	bytes, err := ioutil.ReadFile(a.ConfigFile)
	if err != nil {
		return err
	}
	err = json.Unmarshal(bytes, a)
	if err != nil {
		return err
	}
	return nil
}

//LoadComponents loads components from path
func (a *App) LoadComponents() error {
	a.Components = make(map[string]Component)
	files, err := ioutil.ReadDir(a.ComponentsPath)
	if err != nil {
		return err
	}
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
		log.Println("Rendering", page.Route)
		html, err := page.Render(a.Components, r.URL.Path)
		if err != nil {
			log.Println("ERROR:", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
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
	//Add routes for components and data
	for _, comp := range a.Components {
		comp.AddRoutes(a.Mux)
	}
	return nil
}
