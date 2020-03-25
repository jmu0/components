package main

import (
	"log"
	"os"
	"os/exec"
	"strings"
)

func runWebpack() {
	var cmd []string
	cmd = append(cmd, "webpack")
	if app.Debug == true {
		cmd = append(cmd, "--mode=development")
	} else {
		cmd = append(cmd, "--mode=production")
	}
	cmd = append(cmd, "--entry=./static/js/index.js")
	var outfile string
	if app.Title == "" {
		outfile = "app.js"
	} else {
		outfile = app.Title + ".js"
	}
	outPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	cmd = append(cmd, "--output-filename="+outfile)
	cmd = append(cmd, "--output-path="+outPath+"/static/js")
	log.Println("Webpack command: npx", strings.Join(cmd, " "))
	out, err := exec.Command("npx", cmd...).Output()
	if err != nil {
		log.Fatal("ERROR:", err, "OUTPUT:", string(out))
	}
	log.Println("Webpack output:", string(out))
}
