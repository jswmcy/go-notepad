package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	port    = ":3000"
	dataDir = "./data"
)

type Note struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

func main() {
	os.MkdirAll(dataDir, 0755)
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/list", listHandler)
	http.HandleFunc("/load", loadHandler)
	http.HandleFunc("/save", saveHandler)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	fmt.Printf("go-notepad running on http://localhost%s\n", port)
	http.ListenAndServe(port, nil)
}

// 所有路径都返回首页，由前端根据路径判断打开哪个笔记本
func indexHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "static/index.html")
}

func listHandler(w http.ResponseWriter, r *http.Request) {
	files, _ := os.ReadDir(dataDir)
	var notes []Note
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".txt") {
			name := strings.TrimSuffix(f.Name(), ".txt")
			notes = append(notes, Note{Name: name, Path: f.Name()})
		}
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(notes)
}

func loadHandler(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if name == "" {
		name = "default"
	}
	path := filepath.Join(dataDir, name+".txt")
	data, _ := os.ReadFile(path)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write(data)
}

func saveHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	content := r.FormValue("content")
	if name == "" {
		name = "default"
	}
	path := filepath.Join(dataDir, name+".txt")
	os.MkdirAll(filepath.Dir(path), 0755)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Write([]byte("ok"))
}

func _() { _ = io.EOF }
