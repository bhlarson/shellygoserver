// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"text/template"
	"time"

	"github.com/gorilla/websocket"
)

// var config map[string]interface{}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

var clients = make(map[*websocket.Conn]bool) // connected clients
var broadcast = make(chan Message)           // broadcast channel
// Configure the upgrader

// Define our message object
type Message struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Message  string `json:"message"`
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

// {"name":"Wall High", "address":"192.168.0.115", "device":"shelly2.5","user":"brad", "password":"unstable" },

type ShellyDevice struct {
	name     string
	address  string
	relay    int
	device   string
	user     string
	password string
}

var config []ShellyDevice

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

func ApartmentOn() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("ApartmentOn\n")
	})
}

func ApartmentOff() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("ApartmentOff\n")
	})
}

func PorchOn() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("PorchOn\n")
	})
}

func PorchOff() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("PorchOff\n")
	})
}

func FileServer() http.Handler {
	return http.FileServer(http.Dir("./public")) // New code
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte
}

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *Client) readPump() {
	defer func() {
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
	}
}

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued chat messages to the current websocket message.
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func WebSocket() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println(err)
			return
		}
		client := &Client{conn: conn, send: make(chan []byte, 256)}

		// Allow collection of memory referenced by the caller by doing all work in
		// new goroutines.
		go client.writePump()
		go client.readPump()
	})
}

// func ApartmentToggle() (w http.ResponseWriter, r *http.Request) {
func ApartmentToggle() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var getstr = fmt.Sprintf("http://%s:%s@192.168.1.92/relay/0?turn=toggle", config[0].user, config[0].password)

		resp, err := http.Get(getstr)
		if err != nil {
			// handle error
		} else {
			defer resp.Body.Close()
		}
	})
}

func PorchToggle() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var getstr = fmt.Sprintf("http://%s:%s@192.168.1.93/relay/0?turn=toggle", config[0].user, config[0].password)

		resp, err := http.Get(getstr)
		if err != nil {
			// handle error
		} else {
			defer resp.Body.Close()
		}
	})
}

func WallHighToggle() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var getstr = fmt.Sprintf("http://%s:%s@192.168.1.198/relay/0?turn=toggle", config[0].user, config[0].password)

		resp, err := http.Get(getstr)
		if err != nil {
			// handle error
		} else {
			defer resp.Body.Close()
		}
	})
}
func WallLowToggle() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var getstr = fmt.Sprintf("http://%s:%s@192.168.1.198/relay/1?turn=toggle", config[0].user, config[0].password)

		resp, err := http.Get(getstr)
		if err != nil {
			// handle error
		} else {
			defer resp.Body.Close()
		}
	})
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

func EventHandler(s string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf(s)
	})
}

func serveHTTP(port int, errs chan<- error) {

	mux := http.NewServeMux()

	mux.Handle("/event/apartment/on", ApartmentOn())
	mux.Handle("/event/apartment/off", ApartmentOff())
	mux.Handle("/event/porch/on", PorchOn())
	mux.Handle("/event/porch/off", PorchOff())
	mux.Handle("/event/wallhigh/on", EventHandler("Wall High On\n"))
	mux.Handle("/event/wallhigh/off", EventHandler("Wall High Off\n"))
	mux.Handle("/event/walllow/on", EventHandler("Wall Low On\n"))
	mux.Handle("/event/walllow/off", EventHandler("Wall Low Off\n"))

	fmt.Printf("Starting server at port %d\n", port)
	var servestr = fmt.Sprintf(":%d", port)
	errs <- http.ListenAndServe(servestr, mux)
}

func serveHTTPS(port int, errs chan<- error) {
	file, _ := ioutil.ReadFile("config.json")
	if err := json.Unmarshal(file, &config); err != nil {
		fmt.Println(err)
	}

	mux := http.NewServeMux()
	mux.Handle("/", FileServer())   // New code
	mux.Handle("/wss", WebSocket()) // New code
	mux.Handle("/shelly/apartment/toggle", ApartmentToggle())
	mux.Handle("/shelly/porch/toggle", PorchToggle())
	mux.Handle("/shelly/porch/toggle_high", WallHighToggle())
	mux.Handle("/shelly/porch/toggle_low", WallLowToggle())
	fmt.Printf("Starting server at port %d\n", port)
	var servestr = fmt.Sprintf(":%d", port)
	errs <- http.ListenAndServeTLS(servestr, "cert.pem", "privkey.pem", mux)
}

func main() {
	var port int = 7863
	var tlsport int = 7862

	file, _ := ioutil.ReadFile("config.json")
	err := json.Unmarshal(file, &config)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(config)

	errs := make(chan error, 1)  // a channel for errors
	go serveHTTP(port, errs)     // start the http server in a thread
	go serveHTTPS(tlsport, errs) // start the https server in a thread
	log.Fatal(<-errs)            // block until one of the servers writes an error
}
