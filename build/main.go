package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/jmu0/components"
	"github.com/tdewolff/minify"
	"github.com/tdewolff/minify/js"
)

func main() {
	var content string
	var err error
	if len(os.Args) == 1 {
		printHelp()
		return
	}
	switch os.Args[1] {
	case "less":
		app := loadApp()
		var i, j int
		outPath := "static/css/components.less"
		mainPath := "main.less"
		if len(os.Args) > 2 {
			outPath = os.Args[2]
		}
		if len(os.Args) == 4 {
			mainPath = os.Args[3]
		}
		backCount := len(strings.Split(outPath, "/")) - 1
		content = "// main: " + mainPath + "\n"
		for _, cmp := range app.Components {
			if len(cmp.LessFiles) > 0 {
				for i = 0; i < len(cmp.LessFiles); i++ {
					content += "@import \""
					for j = 0; j < backCount; j++ {
						content += "../"
					}
					fmt.Println("Adding", cmp.LessFiles[i])
					content += cmp.LessFiles[i] + "\";\n"

				}
			}
		}
		err := ioutil.WriteFile(outPath, []byte(content), 0770)
		if err != nil {
			fmt.Println("ERROR:", err)
		}
	case "js":
		app := loadApp()
		outPath := "static/js/"
		var debug = false
		var j int
		if len(os.Args) > 2 {
			if os.Args[2] == "debug" {
				debug = true
			} else {
				outPath = os.Args[2]
				if outPath[:len(outPath)-1] != "/" {
					outPath += "/"
				}
			}
		}
		if len(os.Args) == 4 {
			if os.Args[3] == "debug" {
				debug = true
			}
		}

		backCount := len(strings.Split(outPath, "/")) - 1
		if debug {
			//create symbolic links to files
			var linkPath string
			var sourcePath string
			for _, cmp := range app.Components {
				for i := range cmp.JsFiles {
					if filepath.Base(cmp.JsFiles[i]) == cmp.Name+".js" {
						linkPath = outPath + filepath.Base(cmp.JsFiles[i])
					} else {
						linkPath = outPath + cmp.Name + "." + filepath.Base(cmp.JsFiles[i])
					}
					sourcePath = ""
					for j = 0; j < backCount; j++ {
						sourcePath += "../"
					}
					sourcePath += cmp.JsFiles[i]
					if _, err := os.Lstat(linkPath); err == nil {
						os.Remove(linkPath)
					}
					err = os.Symlink(sourcePath, linkPath)
					if err != nil {
						fmt.Println("ERROR:", err)
					}
				}
			}
		} else {
			//concatinate js files into single file
			content = ""
			for _, cmp := range app.Components {
				if len(cmp.JsFiles) > 0 {
					content += "\n//component " + cmp.Name + "\n\n"
				}
				for i := range cmp.JsFiles {
					fileContent, err := ioutil.ReadFile(cmp.JsFiles[i])
					if err != nil {
						fmt.Println("ERROR:", err)
						continue
					}
					content += string(fileContent) + "\n\n"
				}
			}
			var outfile string
			if app.Title == "" {
				outfile = "app.js"
			} else {
				outfile = app.Title + ".js"
			}
			m := minify.New()
			m.AddFunc("text/javascript", js.Minify)
			minified, err := m.String("text/javascript", content)
			if err != nil {
				minified = content
				fmt.Println("ERROR:", err)
			}
			err = ioutil.WriteFile(outPath+outfile, []byte(minified), 0770)
			if err != nil {
				fmt.Println("ERROR:", err)
			}
		}
	case "run":
		defer func() {
			log.Println("DEFERRING...")
		}()
		run()
	default:
		printHelp()
	}
}
func printHelp() {
	fmt.Print("Invalid Arguments. Usage:\n\n")
	fmt.Print("Build .less import file for all components: \n\tbuild less [<outfile>] [<mainfile>]\n\n")
	fmt.Print("Build .js file from components: \n\tbuild js [outfile | debug] [debug]\n\n")
	fmt.Print("Run development server: \n\tbuild run\n\n")
}
func loadApp() components.App {
	var err error
	var app components.App
	conf := "app.json" //TODO: load from yaml file
	if _, err := os.Stat("app.yml"); err == nil {
		conf = "app.yml"
	}
	app = components.App{
		ConfigFile: conf,
	}
	err = app.LoadConfig()
	if err != nil {
		fmt.Println("ERROR LoadConfig:", err)
		os.Exit(1)

	}
	err = app.LoadComponents()

	if err != nil {
		fmt.Println("ERROR LoadComponents:", err)
		os.Exit(1)
	}
	return app
}
