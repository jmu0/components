package main

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/radovskyb/watcher"
)

var cmd *exec.Cmd

func run() {
	go socket()
	build()
	start()
	watch()
}

func build() {
	log.Println("Building app...")
	cmd = exec.Command("go", "build", "-o", "app")
	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}
}

func start() {
	cmd = exec.Command("./app")
	var stdBuffer bytes.Buffer
	mw := io.MultiWriter(os.Stdout, &stdBuffer)
	cmd.Stdout = mw
	cmd.Stderr = mw
	log.Println("Starting app...")
	if err := cmd.Start(); err != nil {
		log.Fatal("ERROR:", err)
	}
	log.Println(stdBuffer.String())
}

func kill() {
	log.Println("Killing app...")
	if err := cmd.Process.Kill(); err != nil {
		log.Fatal("failed to kill process: ", err)
	}
}

func checkExtension(file string) bool {
	switch filepath.Ext(file) {
	case ".go":
		return true
	case ".html":
		return true
	case ".js":
		return true
	case ".css":
		return true
	case ".yml":
		return true
	case ".scss":
		return true
	default:
		return false
	}
}

func watch() {
	w := watcher.New()

	go func() {
		for {
			select {
			case event := <-w.Event:
				wd, err := os.Getwd()
				if err != nil {
					wd = ""
				}
				if checkExtension(event.Name()) {
					log.Println("Change detected:", strings.Replace(event.Path, wd, "", -1))
					if filepath.Ext(event.Name()) == ".go" {
						kill()
						build()
						start()
					} else if filepath.Ext(event.Name()) == ".scss" {
						buildSass()
					} else if filepath.Ext(event.Name()) == ".js" && app.Debug == false && app.Webpack == true {
						app.RunWebpack()
					}
					reload()
				}
			case err := <-w.Error:
				log.Fatalln(err)
			case <-w.Closed:
				return
			}
		}
	}()

	// Watch this folder for changes.
	if err := w.AddRecursive("."); err != nil {
		log.Fatalln(err)
	}
	// Start the watching process - it'll check for changes every 100ms.
	log.Println("Watching filesystem...")
	if err := w.Start(time.Millisecond * 100); err != nil {
		log.Fatalln(err)
	}
}

func socket() {
	//Connect websocket
	hub = newHub()
	go hub.run()
	mx := http.NewServeMux()
	mx.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})
	log.Fatal(http.ListenAndServe(":9876", mx))
}

func reload() {
	log.Println("Reloading browser...")
	SendSocketMessage([]byte("reload"))
}

func buildSass() {
	log.Println("Compiling sass:", "sass", app.MainSassFile+":"+app.MainCSSFile)
	cmd = exec.Command("sass", app.MainSassFile+":"+app.MainCSSFile)
	var stdBuffer bytes.Buffer
	mw := io.MultiWriter(os.Stdout, &stdBuffer)
	cmd.Stdout = mw
	cmd.Stderr = mw
	if err := cmd.Start(); err != nil {
		log.Println("Sass Error:", err)
	}
	log.Println(stdBuffer.String())
}
