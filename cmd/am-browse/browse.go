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

//go:embed frontend/icons/copy.svg
//go:embed frontend/index.html
var f0 embed.FS

func main() {
	var port int
	flag.IntVar(&port, "port", 8081, "Port to run this server on")
	var am string
	flag.StringVar(&am, "am", "http://localhost:8080", "Base URL for access manager")
	var key string
	flag.StringVar(&key, "key", "", "Private key for server identity")
	flag.Parse()

	f, err := f0.Open("frontend/index.html")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	all, err := io.ReadAll(f)
	if err != nil {
		return
	}
	tmpl := template.Must(template.New("frontend/index").Parse(string(all)))

	// Serve the web page
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		err := tmpl.Execute(w, nil)
		if err != nil {
			return
		}
	})

	http.HandleFunc("/api/details", loginHandler(am))
	http.HandleFunc("/api/dataCredential", dataHandler(am))
	http.HandleFunc("/icons/{name}", iconHandler)

	// Start the server
	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
		return
	}
}

func iconHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFileFS(w, r, f0, fmt.Sprintf("frontend/icons/%s", r.PathValue("name")))
}

func loginHandler(baseUrl string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
			return
		}

		details, err := getDetails(baseUrl, r.FormValue("path"), r.FormValue(("caller_id")))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(details))
	}
}

func dataHandler(baseUrl string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
			return
		}

		path := r.FormValue("path")
		caller_id := r.FormValue("caller_id")
		body, err := fetch(baseUrl, "/datasetcredential", "path", path, "caller_id", caller_id)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		response := struct {
			Url        string // a physical location for the credential in device specific format
			Info       string // a human readable description of the credential
			Credential string // the signed credential itself, also device specific format
			Error      struct {
				Error   int    `json:"error"`
				Message string `json:"message"`
				Global  any    `json:"global"`
			}
		}{}
		if err := json.Unmarshal(body, &response); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if response.Error.Error != 0 {
			http.Error(w, fmt.Sprintf(`error getting credential "%s": %s`, path, response.Error.Message), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response.Credential))
	}
}

type ServerError error

func getDetails(baseUrl string, path string, ourCredential string) (string, error) {
	body, err := fetch(baseUrl, "/details", "path", path, "include_children", "true", "caller_id", strings.TrimSpace(ourCredential))
	if err != nil {
		return "", err
	}
	type NodeDetails struct {
		Path           string           `json:"path"`
		Roles          []AppliedRole    `json:"roles"`
		InheritedRoles []AppliedRole    `json:"inherited_roles"`
		Aces           []ACE            `json:"aces"`
		InheritedAces  []ACE            `json:"inherited_aces"`
		Annotations    []UserAnnotation `json:"annotations"`
		IsDirectory    bool             `json:"is_directory"`
	}
	details := struct {
		Details  NodeDetails `json:"details"`
		Children []string    `json:"children"`
		Error    struct {
			Error   int    `json:"error"`
			Message string `json:"message"`
			Global  any    `json:"global"`
		}
	}{}
	if err := json.Unmarshal(body, &details); err != nil {
		return "", fmt.Errorf(`error parsing response: %w`, err)
	}
	if details.Error.Error != 0 {
		return "", fmt.Errorf(`error getting credential "%s": %s`, path, details.Error.Message)
	}
	return string(body), nil
}

func fetch(baseUrl, action string, kv ...string) ([]byte, error) {
	params := url.Values{}
	if len(kv)%2 != 0 {
		return nil, fmt.Errorf(`need even number of optional arguments, got %d`, len(kv))
	}
	for i := 0; i < len(kv); i += 2 {
		params.Add(kv[i], kv[i+1])
	}

	u, _ := url.ParseRequestURI(baseUrl)
	u.Path = action
	u.RawQuery = params.Encode()
	request, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		u.String(),
		http.NoBody,
	)
	if err != nil {
		return nil, ServerError(fmt.Errorf(`error posting request for "%s": %w`, u.String(), err))
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, ServerError(fmt.Errorf(`cannot access "%s": %w`, u.String(), err))
	}
	//goland:noinspection GoUnhandledErrorResult
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(`unexpected status code "%d"`, response.StatusCode)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf(`error reading response from %s: %w`, u.String(), err)
	}
	return body, nil
}

type Operation int

type ACE struct {
	Op          Operation `json:"op"`
	Local       bool      `json:"local"`
	Acls        []ACL     `json:"acls"`
	Unique      int64     `json:"unique,string"`
	Version     int64     `json:"version,string"`
	StartMillis int64     `json:"startMillis,string"`
	EndMillis   int64     `json:"endMillis,string"`
}

type ACL struct {
	Roles []string `json:"roles"`
}

type AppliedRole struct {
	Role        string `json:"role"`
	Tag         string `json:"tag"`
	Unique      int64  `json:"unique,string"`
	Version     int64  `json:"version,string"`
	StartMillis int64  `json:"startMillis,string"`
	EndMillis   int64  `json:"endMillis,string"`
}

type UserAnnotation struct {
	Data        string `json:"data"`
	Tag         string `json:"tag"`
	Unique      int64  `json:"unique,string"`
	Version     int64  `json:"version,string"`
	StartMillis int64  `json:"startMillis,string"`
	EndMillis   int64  `json:"endMillis,string"`
}

func (op *Operation) UnmarshalJSON(b []byte) error {
	z, ok := Operation_value[strings.Trim(strings.ToUpper(string(b)), `"`)]
	if !ok {
		return fmt.Errorf("invalid operation %q", b)
	}
	*op = Operation(z)
	return nil
}

var Operation_value = map[string]int32{
	"INVALID":   0,
	"READ":      1,
	"WRITE":     2,
	"VIEW":      3,
	"ADMIN":     4,
	"USEROLE":   5,
	"APPLYROLE": 6,
	"VOUCHFOR":  7,
}
