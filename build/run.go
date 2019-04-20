package main

import (
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
	// netstat -ltnp | grep 8282

	// gofiles, err := filepath.Glob("*.go")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// args := []string{"run"}
	// args = append(args, gofiles...)
	// cmd = exec.Command("go", args...)
	cmd = exec.Command("./app")
	//kill child processes: cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// stdout, err := cmd.StdoutPipe()
	// if err != nil {
	// 	log.Println("App error:", err)
	// }

	log.Println("Starting app...")
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	// in := bufio.NewScanner(stdout)
	// for in.Scan() {
	// 	log.Println("App:", in.Text())
	// }
	// if err := in.Err(); err != nil {
	// 	log.Fatal(err)
	// }

}

func kill() {
	// syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)

	log.Println("Killing app...")
	if err := cmd.Process.Kill(); err != nil {
		log.Fatal("failed to kill process: ", err)
	}
}

func watch() {
	w := watcher.New()

	go func() {
		for {
			select {
			case event := <-w.Event:
				// wd, err := dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
				wd, err := os.Getwd()
				if err != nil {
					wd = ""
				}
				log.Println("Change detected:", strings.Replace(event.Path, wd, "", -1))
				if event.Name() != "app" {
					kill()
					if filepath.Ext(event.Name()) == ".go" {
						build()
					}
					start()
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
