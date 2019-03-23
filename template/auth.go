package main

import (
	"errors"
	"log"
	"net/http"

	"git.muysers.nl/jmu0/jwt"
)

func authenticate(username, password string) (map[string]string, error) {
	var ret = make(map[string]string)
	if username == "jos" && password == "123" {
		ret["name"] = "Jos Muysers"
		ret["keys"] = "name=jos"
		ret["authenticated"] = "true"
	} else {
		log.Println("Auth failed: invalid password")
		return ret, errors.New("Invalid Password")
	}
	log.Println("Authenticated:", ret["name"])
	return ret, nil
}

func handleAuth(w http.ResponseWriter, r *http.Request) {
	err := jwt.HandleAuth(w, r, authenticate)
	if err != nil {
		log.Println("Auth failed:", err)
	}
}
