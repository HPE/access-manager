/*
 * SPDX-FileCopyrightText:  Copyright Hewlett Packard Enterprise Development LP
 */

package main

import (
	"context"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type User struct {
	Name     string
	Password string
}

func main() {
	// Sample users. In the real world, these would be in a user auth system
	users := []User{
		{"Alice", "apple,tree"},
		{"Bob", "blue,sky"},
		{"Charlie", "cat,mouse"},
		{"David", "dog,house"},
		{"Eve", "earth,wind"},
	}
	var port int
	flag.IntVar(&port, "port", 8082, "Port to run this server on")
	var am string
	flag.StringVar(&am, "am", "http://localhost:8080", "Base URL for access manager")
	var key string
	flag.StringVar(&key, "key", "", "Private key for server identity")
	flag.Parse()

	f, err := f0.Open("frontend/index.html")
	if err != nil {
		panic(err)
	}
	htmlTemplate, err := io.ReadAll(f)
	if err != nil {
		panic(err)
	}
	_ = f.Close()

	// Serve the web page
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tmpl := template.Must(template.New("frontend/index").Parse(string(htmlTemplate)))
		err := tmpl.Execute(w, users)
		if err != nil {
			return
		}
	})

	http.HandleFunc("/api/login", loginHandler(users, am))
	http.HandleFunc("/icons/{name}", iconHandler)

	// Start the server
	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
		return
	}
}

//go:embed frontend/icons/copy.svg
//go:embed frontend/index.html
var f0 embed.FS

func iconHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFileFS(w, r, f0, fmt.Sprintf("frontend/icons/%s", r.PathValue("name")))
}

func loginHandler(users []User, baseUrl string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
			return
		}

		b, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Error reading request body", http.StatusInternalServerError)
			return
		}
		args := struct {
			User     string `json:"user"`
			Password string `json:"password"`
			CallerID string `json:"caller_id"`
		}{}
		if err := json.Unmarshal(b, &args); err != nil {
			return
		}
		if !checkUser(users, args.User, args.Password) {
			http.Error(w, "Invalid username or password", http.StatusUnauthorized)
			return
		}
		credential, err := getUserCredential(baseUrl, "am://user/yoyodyne/"+args.User, args.CallerID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Example: Handle login logic here
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(credential))
	}
}

func checkUser(users []User, user string, password string) bool {
	for _, u := range users {
		if strings.ToLower(u.Name) == strings.ToLower(user) && u.Password == password {
			return true
		}
	}
	return false
}

func getUserCredential(baseUrl string, path string, ourCredential string) (string, error) {
	params := url.Values{}
	params.Add("path", path)
	params.Add("caller_id", strings.TrimSpace(ourCredential))
	u, _ := url.ParseRequestURI(baseUrl + "/credential")
	u.RawQuery = params.Encode()
	request, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		u.String(),
		http.NoBody,
	)
	if err != nil {
		return "", fmt.Errorf(`error posting request for "%s": %w`, u.String(), err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", fmt.Errorf(`cannot access "%s": %w`, u.String(), err)
	}
	//goland:noinspection GoUnhandledErrorResult
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf(`unexpected status code "%d"`, response.StatusCode)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf(`error reading response from %s: %w`, u.String(), err)
	}
	details := struct {
		Credential string `json:"credential"`
		Error      struct {
			Error   int    `json:"error"`
			Message string `json:"message"`
			Global  any    `json:"global"`
		}
	}{}
	if err = json.Unmarshal(body, &details); err != nil {
		return "", fmt.Errorf(`error parsing response: %w`, err)
	}
	if details.Error.Error != 0 {
		return "", fmt.Errorf(`error getting credential "%s": %s`, path, details.Error.Message)
	}
	return details.Credential, nil
}
