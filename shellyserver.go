// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

/*
type Creds struct {
	user     string `json:"user"`
	password string `json:"password"`
}

var creds Creds
*/

var creds map[string]interface{}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/hello" {
		http.Error(w, "404 not found.", http.StatusNotFound)
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Method is not supported.", http.StatusNotFound)
		return
	}

	fmt.Fprintf(w, "Hello!")
}

func formHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		fmt.Fprintf(w, "ParseForm() err: %v", err)
		return
	}
	fmt.Fprintf(w, "POST request successful")
	name := r.FormValue("name")
	address := r.FormValue("address")

	fmt.Fprintf(w, "Name = %s\n", name)
	fmt.Fprintf(w, "Address = %s\n", address)
}

func ApartmentOn(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("ApartmentOn\n")
}

func ApartmentOff(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("ApartmentOff\n")
}

func ApartmentToggle(w http.ResponseWriter, r *http.Request) {
	var getstr = fmt.Sprintf("http://%s:%s@192.168.1.226/relay/0?turn=toggle", creds["user"], creds["password"])
	resp, err := http.Get(getstr)
	if err != nil {
		// handle error
	}
	defer resp.Body.Close()
}

func main() {
	var port int = 8080

	file, _ := ioutil.ReadFile("creds.json")
	if err := json.Unmarshal(file, &creds); err != nil {
		fmt.Println(err)
	}

	fileServer := http.FileServer(http.Dir("./public")) // New code
	http.Handle("/", fileServer)                        // New code
	http.HandleFunc("/hello", helloHandler)             // Update this line of code
	http.HandleFunc("/form", formHandler)
	http.HandleFunc("/shelly/apartment/on", ApartmentOn)
	http.HandleFunc("/shelly/apartment/off", ApartmentOff)
	http.HandleFunc("/shelly/apartment/toggle", ApartmentToggle)

	fmt.Printf("Starting server at port %d\n", port)

	var servestr = fmt.Sprintf(":%d", port)
	if err := http.ListenAndServe(servestr, nil); err != nil {
		log.Fatal(err)
	}
}
