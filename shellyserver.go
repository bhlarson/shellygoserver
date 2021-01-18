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
	"text/template"

	"github.com/gorilla/websocket"
)

/*
type Creds struct {
	user     string `json:"user"`
	password string `json:"password"`
}

var creds Creds
*/

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func wsEndpoint(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }

	// upgrade this connection to a WebSocket
	// connection
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
	}

	log.Println("Client Connected")
	err = ws.WriteMessage(1, []byte("Hi Client!"))
	if err != nil {
		log.Println(err)
	}
	// listen indefinitely for new messages coming
	// through on our WebSocket connection
	reader(ws)
}

// define a reader which will listen for
// new messages being sent to our WebSocket
// endpoint
func reader(conn *websocket.Conn) {
	for {
		// read in a message
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		}
		// print out that message for clarity
		fmt.Println(string(p))

		if err := conn.WriteMessage(messageType, p); err != nil {
			log.Println(err)
			return
		}

	}
}

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
	var getstr = fmt.Sprintf("http://%s:%s@192.168.1.92/relay/0?turn=toggle", creds["user"], creds["password"])

	resp, err := http.Get(getstr)
	if err != nil {
		// handle error
	} else {
		defer resp.Body.Close()
	}
}

func PorchToggle(w http.ResponseWriter, r *http.Request) {
	var getstr = fmt.Sprintf("http://%s:%s@192.168.1.93/relay/0?turn=toggle", creds["user"], creds["password"])

	resp, err := http.Get(getstr)
	if err != nil {
		// handle error
	} else {
		defer resp.Body.Close()
	}
}

func login(w http.ResponseWriter, r *http.Request) {
	fmt.Println("method:", r.Method) //get request method
	if r.Method == "GET" {
		t, _ := template.ParseFiles("login.gtpl")
		t.Execute(w, nil)
	} else {
		r.ParseForm()
		// logic part of log in
		fmt.Println("username:", r.Form["username"])
		fmt.Println("password:", r.Form["password"])
	}
}

func main() {
	var port int = 9000

	file, _ := ioutil.ReadFile("creds.json")
	if err := json.Unmarshal(file, &creds); err != nil {
		fmt.Println(err)
	}

	fileServer := http.FileServer(http.Dir("./public")) // New code
	http.Handle("/", fileServer)                        // New code
	http.HandleFunc("/login", login)
	http.HandleFunc("/ws", wsEndpoint)
	http.HandleFunc("/hello", helloHandler) // Update this line of code
	http.HandleFunc("/form", formHandler)
	http.HandleFunc("/shelly/apartment/on", ApartmentOn)
	http.HandleFunc("/shelly/apartment/off", ApartmentOff)
	http.HandleFunc("/shelly/apartment/toggle", ApartmentToggle)
	http.HandleFunc("/shelly/porch/toggle", PorchToggle)

	fmt.Printf("Starting server at port %d\n", port)

	var servestr = fmt.Sprintf(":%d", port)
	//if err := http.ListenAndServe(servestr, nil); err != nil {
	//	log.Fatal(err)
	//}

	log.Fatal(http.ListenAndServeTLS(servestr, "cert.pem", "privkey.pem", nil))
}
