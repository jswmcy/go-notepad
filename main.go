package main

import (
	"fmt"
	"net/http"
	"os"
)

func main() {
	os.MkdirAll("./data", 0755)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, "static/index.html")
	})

	http.HandleFunc("/load", func(w http.ResponseWriter, r *http.Request) {
		data, _ := os.ReadFile("./data/note.txt")
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write(data)
	})

	http.HandleFunc("/save", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", 405)
			return
		}
		content := r.FormValue("content")
		if err := os.WriteFile("./data/note.txt", []byte(content), 0644); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Write([]byte("ok"))
	})

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	fmt.Printf("go-notepad running on http://localhost:3000\n")
	http.ListenAndServe(":3000", nil)
}
