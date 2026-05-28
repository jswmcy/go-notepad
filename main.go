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
	version = "v1.2.1"
	buildDate = "2026-05-28"
)

type Note struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

func main() {
	os.MkdirAll(dataDir, 0755)
	
	// 文本笔记 API
	http.HandleFunc("/list", listHandler)
	http.HandleFunc("/load", loadHandler)
	http.HandleFunc("/save", saveHandler)
	
	// 图片笔记相关 API
	http.HandleFunc("/upload-image", uploadImageHandler)
	http.HandleFunc("/list-images", listImagesHandler)
	http.HandleFunc("/image-note", imageNoteHandler)
	http.HandleFunc("/delete-image", deleteImageHandler)
	
	// 版本信息
	http.HandleFunc("/version", versionHandler)
	
	// 静态文件服务：static/ 目录
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	// 数据文件服务：data/ 目录（用于图片访问）
	http.Handle("/data/", http.StripPrefix("/data/", http.FileServer(http.Dir("data"))))
	
	// 最后注册通配符首页
	http.HandleFunc("/", indexHandler)
	
	fmt.Printf("go-notepad running on http://localhost%s\n", port)
	http.ListenAndServe(port, nil)
}

// 版本信息处理器
func versionHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(map[string]string{
		"version":   version,
		"buildDate": buildDate,
		"app":       "go-notepad",
		"status":    "running",
	})
}

// 所有路径都返回首页，由前端根据路径判断打开哪个笔记本
func indexHandler(w http.ResponseWriter, r *http.Request) {
	// /version 页面返回版本信息
	if r.URL.Path == "/version" {
		versionHandler(w, r)
		return
	}
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
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if name == "" {
		name = "default"
	}
	body, _ := io.ReadAll(r.Body)
	defer r.Body.Close()
	content := string(body)
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
	originalFilename := header.Filename
	ext := strings.ToLower(filepath.Ext(originalFilename))
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

// 图片笔记块类型
type ImageNoteBlock struct {
	Type     string `json:"type"`     // "image" 或 "text"
	Content  string `json:"content"`  // 图片文件名 或 文字内容
	ID       string `json:"id"`       // 块唯一ID（时间戳+随机）
	Created  string `json:"created"`  // 创建时间
}

// 图片笔记文档
type ImageNoteDoc struct {
	Name   string           `json:"name"`
	Blocks []ImageNoteBlock `json:"blocks"`
	Version int             `json:"version"`
}

// GET /image-note?name=笔记本名
func getImageNoteHandler(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if name == "" {
		http.Error(w, "Missing name parameter", 400)
		return
	}
	
	docPath := filepath.Join(dataDir, name+"_image_note.json")
	data, err := os.ReadFile(docPath)
	if err != nil {
		// 文件不存在返回空文档
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ImageNoteDoc{
			Name:   name,
			Blocks: []ImageNoteBlock{},
			Version: 1,
		})
		return
	}
	
	var doc ImageNoteDoc
	err = json.Unmarshal(data, &doc)
	if err != nil {
		http.Error(w, "Invalid image note format", 500)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(doc)
}

// POST /image-note
func saveImageNoteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	
	var doc ImageNoteDoc
	err := json.NewDecoder(r.Body).Decode(&doc)
	if err != nil {
		http.Error(w, "Invalid JSON", 400)
		return
	}
	
	if doc.Name == "" {
		http.Error(w, "Missing name field", 400)
		return
	}
	
	docPath := filepath.Join(dataDir, doc.Name+"_image_note.json")
	
	// 确保目录存在
	os.MkdirAll(filepath.Dir(docPath), 0755)
	
	// 保存文档
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		http.Error(w, "Failed to marshal document", 500)
		return
	}
	
	err = os.WriteFile(docPath, data, 0644)
	if err != nil {
		http.Error(w, "Failed to save document", 500)
		return
	}
	
	w.Write([]byte("ok"))
}

// 统一图片笔记处理器
func imageNoteHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		getImageNoteHandler(w, r)
	case "POST":
		saveImageNoteHandler(w, r)
	default:
		http.Error(w, "Method not allowed", 405)
	}
}

// DELETE /delete-image?name=笔记本名&filename=图片名
func deleteImageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", 405)
		return
	}

	name := strings.TrimSpace(r.URL.Query().Get("name"))
	filename := strings.TrimSpace(r.URL.Query().Get("filename"))
	if name == "" || filename == "" {
		http.Error(w, "Missing name or filename parameter", 400)
		return
	}

	// 安全检查：确保文件名不包含路径穿越
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		http.Error(w, "Invalid filename", 400)
		return
	}

	filePath := filepath.Join(dataDir, name+"_images", filename)
	err := os.Remove(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			w.Write([]byte("deleted")) // 文件不存在也算成功
			return
		}
		http.Error(w, "Failed to delete file: "+err.Error(), 500)
		return
	}

	w.Write([]byte("deleted"))
}
