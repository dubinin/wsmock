package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
	"fmt"
	"strconv"
	"path"
	"os"
)

type Settings struct {
	Login    string    `json:"login"`
	Password string    `json:"password"`
	Handlers []Handler `json:"handlers"`
}

type Handler struct {
	Path    string            `json:"path"`
	Data    string            `json:"data"`
	Headers map[string]string `json:"headers"`
}

type WSMockHandler struct{}

const settingsFile = "settings.json"
const responseFolder = "response"

var config Settings
var HTML = `<html>
	<head>
		<title>WSMock</title>
	</head>
	<body>
		<form action="/admin" method="post">
			<textarea name="settings" rows="40" cols="200">%s</textarea>
			</br>
			<input type="submit" value="Save"/>
		</form>
	</body>
</html>`

func (h *WSMockHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var body []byte
	// Get body
	if strings.HasPrefix(r.URL.Path, "/admin") {
		if r.Method == "GET" {
			w.Header().Set("Content-type", "text/html; charset=utf-8")
			rowSettings, err := ReadSettings()
			if err != nil {
				log.Printf("Can't get %s: %v\n", settingsFile, err)
			} else {
				w.Write([]byte(fmt.Sprintf(HTML, rowSettings)))
			}
		} else if r.Method == "POST" {
			log.Println("Admin try to save settings")
			body = []byte(r.FormValue("settings"))
			err := SetSettings(body)
			if err != nil {
				response := fmt.Sprintf("Can't parse %s: %v\n", settingsFile, err)
				log.Print(response)
				w.Write([]byte(response))
			} else {
				err = WriteSettings(body)
				if err != nil {
					log.Printf("Can't save %s: %v\n", settingsFile, err)
					w.WriteHeader(http.StatusInternalServerError)
				} else {
					w.WriteHeader(http.StatusOK)
				}
			}
		}
		return
	} else {
		// Try to find handler by path
		for _, handler := range config.Handlers {
			if strings.HasPrefix(r.URL.Path, handler.Path) {
				body, err := ioutil.ReadAll(r.Body)
				if err != nil {
					log.Printf("Can't get request body: %v\n", err)
					w.WriteHeader(http.StatusInternalServerError)
				} else {
					// Request logging
					log.Printf("Incoming message for (%s):\n%s\n",
						r.URL.Path, body)
					WriteRequestBody(body, r.URL.Path)
					// Set headers
					for key, value := range handler.Headers {
						w.Header().Set(key, value)
					}
					// Set body
					w.Write([]byte(handler.Data))
					return
				}
			}
		}
	}
	w.WriteHeader(http.StatusNotFound)
}

func main() {
	os.Mkdir(responseFolder, os.ModePerm)
	data, err := ReadSettings()
	if err != nil {
		log.Printf("Can't open %s: %v\n", settingsFile, err)
	} else {
		err = SetSettings(data)
		if err != nil {
			log.Printf("Can't parse %s: %v\n", settingsFile, err)
		}
	}

	s := &http.Server{
		Addr:           ":8080",
		Handler:        &WSMockHandler{},
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	log.Fatal(s.ListenAndServe())
}

func SetSettings(source []byte) error {
	return json.Unmarshal(source, &config)
}

func ReadSettings() ([]byte, error) {
	return ioutil.ReadFile(settingsFile)
}

func WriteSettings(data []byte) error {
	return ioutil.WriteFile(settingsFile, data, 0644)
}

func WriteRequestBody(data []byte, dir string) error {
	if (len(data) == 0) {
		return nil
	}
	return ioutil.WriteFile(path.Join(responseFolder,
		strconv.FormatInt(time.Now().UnixNano(), 10)), data, 0644)
}
