package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	port    = ":3000"
	dataDir = "./data"
)

type Note struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type ImageMetaEntry struct {
	Image string `json:"image"`
	Note  string `json:"note"`
}

func main() {
	os.MkdirAll(dataDir, 0755)
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/list", listHandler)
	http.HandleFunc("/load", loadHandler)
	http.HandleFunc("/save", saveHandler)
	
	// 图片笔记本相关 API
	http.HandleFunc("/upload-image", uploadImageHandler)
	http.HandleFunc("/list-images", listImagesHandler)
	http.HandleFunc("/meta", metaHandler)
	
	// 静态文件服务：static/ 目录
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	// 数据文件服务：data/ 目录（用于图片访问）
	http.Handle("/data/", http.StripPrefix("/data/", http.FileServer(http.Dir("data"))))
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

// ========== 图片笔记本相关 API ==========

// 生成随机文件名
func generateRandomFilename(ext string) string {
	now := time.Now()
	dateStr := now.Format("20060102_150405")
	
	buf := make([]byte, 3) // 6位hex
	rand.Read(buf)
	randomStr := hex.EncodeToString(buf)
	
	return fmt.Sprintf("%s_%s.%s", dateStr, randomStr, ext)
}

// POST /upload-image?name=笔记本名
func uploadImageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if name == "" {
		http.Error(w, "Missing name parameter", 400)
		return
	}
	
	// 限制10MB
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20) // 10MB
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		http.Error(w, "File too large (max 10MB)", http.StatusRequestEntityTooLarge)
		return
	}
	
	file, header, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "No image file provided: " + err.Error(), 400)
		return
	}
	defer file.Close()
	
	// 检查文件类型（从文件名判断扩展名，MIME 头不可靠）
	filename := header.Filename
	ext := strings.ToLower(filepath.Ext(filename))
	allowedExts := map[string]string{
		".png":  "png",
		".jpg":  "jpg",
		".jpeg": "jpg",
		".gif":  "gif",
		".webp": "webp",
	}
	outExt, ok := allowedExts[ext]
	if !ok {
		// 兜底：从 MIME 类型推断
		mimeType := header.Header.Get("Content-Type")
		switch mimeType {
		case "image/png":
			outExt = "png"
		case "image/jpeg":
			outExt = "jpg"
		case "image/gif":
			outExt = "gif"
		case "image/webp":
			outExt = "webp"
		default:
			http.Error(w, "Unsupported image type. Allowed: PNG, JPEG, GIF, WebP", 400)
			return
		}
	}
	
	// 创建图片目录
	imageDir := filepath.Join(dataDir, name+"_images")
	err = os.MkdirAll(imageDir, 0755)
	if err != nil {
		http.Error(w, "Failed to create directory", 500)
		return
	}
	
	// 生成文件名
	filename := generateRandomFilename(outExt)
	filePath := filepath.Join(imageDir, filename)
	
	// 保存文件
	dst, err := os.Create(filePath)
	if err != nil {
		http.Error(w, "Failed to save file: " + err.Error(), 500)
		return
	}
	defer dst.Close()
	
	_, err = io.Copy(dst, file)
	if err != nil {
		http.Error(w, "Failed to save file: " + err.Error(), 500)
		return
	}
	
	// 返回结果
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"path":     "/data/" + name + "_images/" + filename,
		"filename": filename,
	})
}

// GET /list-images?name=笔记本名
func listImagesHandler(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if name == "" {
		http.Error(w, "Missing name parameter", 400)
		return
	}
	
	imageDir := filepath.Join(dataDir, name+"_images")
	files, err := os.ReadDir(imageDir)
	if err != nil {
		// 目录不存在返回空数组
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]string{})
		return
	}
	
	var imageFiles []string
	for _, f := range files {
		if !f.IsDir() {
			ext := strings.ToLower(filepath.Ext(f.Name()))
			if ext == ".png" || ext == ".jpg" || ext == ".jpeg" || ext == ".gif" || ext == ".webp" {
				imageFiles = append(imageFiles, f.Name())
			}
		}
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(imageFiles)
}

// GET /meta?name=笔记本名
func getMetaHandler(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if name == "" {
		http.Error(w, "Missing name parameter", 400)
		return
	}
	
	metaPath := filepath.Join(dataDir, name+"_images", "meta.json")
	data, err := os.ReadFile(metaPath)
	if err != nil {
		// 文件不存在返回空对象
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]ImageMetaEntry{})
		return
	}
	
	var meta map[string]ImageMetaEntry
	err = json.Unmarshal(data, &meta)
	if err != nil {
		http.Error(w, "Invalid meta.json format", 500)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(meta)
}

// POST /meta
func postMetaHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	
	var req struct {
		Name  string `json:"name"`
		Image string `json:"image"`
		Note  string `json:"note"`
	}
	
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid JSON", 400)
		return
	}
	
	if req.Name == "" || req.Image == "" {
		http.Error(w, "Missing required fields", 400)
		return
	}
	
	metaPath := filepath.Join(dataDir, req.Name+"_images", "meta.json")
	
	// 读取现有meta
	var meta map[string]ImageMetaEntry
	data, err := os.ReadFile(metaPath)
	if err == nil {
		json.Unmarshal(data, &meta)
	}
	
	if meta == nil {
		meta = make(map[string]ImageMetaEntry)
	}
	
	// 更新或添加条目
	meta[req.Image] = ImageMetaEntry{
		Image: req.Image,
		Note:  req.Note,
	}
	
	// 保存meta.json
	newData, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		http.Error(w, "Failed to marshal meta", 500)
		return
	}
	
	// 确保目录存在
	os.MkdirAll(filepath.Dir(metaPath), 0755)
	err = os.WriteFile(metaPath, newData, 0644)
	if err != nil {
		http.Error(w, "Failed to save meta", 500)
		return
	}
	
	w.Write([]byte("ok"))
}

// 统一meta处理器
func metaHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		getMetaHandler(w, r)
	case "POST":
		postMetaHandler(w, r)
	default:
		http.Error(w, "Method not allowed", 405)
	}
}
