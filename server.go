package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"image/gif"
	"image/png"
	"io"
	"log"
	"math/rand/v2"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

type CC_MonitorSize struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type CurrentContent struct {
	Name string `json:"name"`
	Anim bool   `json:"anim"`
}

var mtx = sync.Mutex{}
var contentInitialized bool = false
var serverCurrentContent = CurrentContent{
	Name: "none",
	Anim: false,
}
var MonitorSizes = CC_MonitorSize{
	Width:  79,
	Height: 52,
}
var outputFilePath string = "/imgs/Converted"

func handleImageUpload(w http.ResponseWriter, r *http.Request) {
	// goroutines that have to lock mtx won't do anything untill this section of code will end
	mtx.Lock()
	defer mtx.Unlock()
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodPost {
		http.Error(w, "Not a POST method", http.StatusMethodNotAllowed)
		return
	}
	img, _, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "file forming failed", http.StatusBadRequest)
		return
	}
	defer img.Close()

	buff := make([]byte, 512)
	if _, err := img.Read(buff); err != nil {
		http.Error(w, "error reading file heading", http.StatusInternalServerError)
		return
	}

	// Сбрасываем указатель чтения файла обратно на начало,
	// иначе при сохранении файл запишется без первых 512 байт
	if _, err := img.Seek(0, io.SeekStart); err != nil {
		http.Error(w, "error in io.SeekStart", http.StatusInternalServerError)
		return
	}

	fileType := http.DetectContentType(buff)
	fileFormat := ""

	if fileType != "image/jpeg" && fileType != "image/png" {
		http.Error(w, "Not a png/jpeg file", http.StatusBadRequest)
		return
	} else if fileType == "image/jpeg" {
		fileFormat = ".jpeg"
	} else {
		fileFormat = ".png"
	}

	path := "./imgs/"
	filename := strconv.Itoa(rand.Int()) + fileFormat
	dst, err := os.Create(filepath.Join(path, filename))
	if err != nil {
		http.Error(w, "failed to create the file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, img); err != nil {
		http.Error(w, "Error while writing file", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message":  "File Uploaded",
		"detected": fileType,
	})
	ProcessImage(filepath.Join(path, filename), MonitorSizes.Width, MonitorSizes.Height, outputFilePath+filename)
}

func handleGifUpload(w http.ResponseWriter, r *http.Request) {
	mtx.Lock()
	defer mtx.Unlock()
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodPost {
		http.Error(w, "Not a POST method", http.StatusMethodNotAllowed)
		return
	}
	img, _, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "file forming failed", http.StatusBadRequest)
		return
	}
	defer img.Close()

	buff := make([]byte, 512)
	if _, err := img.Read(buff); err != nil {
		http.Error(w, "error reading file heading", http.StatusInternalServerError)
		return
	}

	// Сбрасываем указатель чтения файла обратно на начало,
	// иначе при сохранении файл запишется без первых 512 байт
	if _, err := img.Seek(0, io.SeekStart); err != nil {
		http.Error(w, "error in io.SeekStart", http.StatusInternalServerError)
		return
	}

	fileType := http.DetectContentType(buff)
	if fileType != "image/gif" {
		http.Error(w, "Not a gif file", http.StatusBadRequest)
		return
	}

	path := "./imgs/"
	filename := strconv.Itoa(rand.Int()) + ".gif"
	gifPath := filepath.Join(path, filename)
	dst, err := os.Create(gifPath)
	if err != nil {
		http.Error(w, "failed to create the file", http.StatusInternalServerError)
		return
	}
	if _, err := io.Copy(dst, img); err != nil {
		dst.Close()
		http.Error(w, "Error while writing file", http.StatusInternalServerError)
		return
	}
	if err := dst.Close(); err != nil {
		http.Error(w, "Error while closing file", http.StatusInternalServerError)
		return
	}

	tmpDir, err := os.MkdirTemp("imgs/", "temp*")
	if err != nil {
		log.Println("Couldn't create temp dir: ", err)
		return
	}
	defer os.RemoveAll(tmpDir)
	if err := DecodeGif(gifPath, tmpDir); err != nil {
		http.Error(w, "Failed to decode GIF", http.StatusBadRequest)
		log.Println("Failed to decode GIF:", err)
		return
	}
	ProcessFrames(tmpDir, MonitorSizes.Width, MonitorSizes.Height, filename, outputFilePath)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message":  "GIF uploaded",
		"detected": fileType,
	})
}

func handleStartPage(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/index.html")
}

func handleMonitorSizeSet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Not a POST method", http.StatusMethodNotAllowed)
		return
	}
	mtx.Lock()
	err := json.NewDecoder(r.Body).Decode(&MonitorSizes)
	if err != nil {
		http.Error(w, "Error decoding monitro sizes json", http.StatusBadRequest)
		log.Println("Failed to decode monitor size json: ", err)
		return
	}
	mtx.Unlock()
	w.WriteHeader(http.StatusOK)

	log.Println("set size to ", MonitorSizes)
}

func handleImageUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Not a POST method", http.StatusMethodNotAllowed)
		return
	}

	var ccCurrentContent CurrentContent

	if err := json.NewDecoder(r.Body).Decode(&ccCurrentContent); err != nil {
		http.Error(w, "Failed to decode current content json", http.StatusBadRequest)
		log.Println("Failed to decode current content json:", err)
		return
	}

	mtx.Lock()
	current := serverCurrentContent
	mtx.Unlock()

	if ccCurrentContent != current {
		w.Header().Set("Content-Type", "application/json")

		if err := json.NewEncoder(w).Encode(current); err != nil {
			log.Println("Failed to send current content: ", err)
			return
		}

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func handleinitContent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Not a POST method", http.StatusMethodNotAllowed)
		return
	}

	var ccCurrentContent CurrentContent

	if err := json.NewDecoder(r.Body).Decode(&ccCurrentContent); err != nil {
		http.Error(w, "Failed to decode json with initial content", http.StatusBadRequest)
		log.Println("Failed to decode json with initial content: ", err)
		return
	}

	mtx.Lock()
	defer mtx.Unlock()

	if contentInitialized {
		http.Error(w, "content was already initialized", http.StatusConflict)
		log.Println("tried to reinit server current content")
		return
	} else {
		contentInitialized = true
		serverCurrentContent = ccCurrentContent
		w.WriteHeader(http.StatusAccepted)
	}
}

func handleGetContentAvaliable(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Not a GET method", http.StatusMethodNotAllowed)
		return
	}

	content, err := os.ReadDir(outputFilePath)
	if err != nil {
		http.Error(w, "failed to get avaliable content", http.StatusBadGateway)
		log.Println("Failed to os.ReadDir for content:", err)
		return
	}

	var filenames []string
	targetExt := ".png"

	for _, entry := range content {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) == targetExt {
			filenames = append(filenames, entry.Name())
		}
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(filenames); err != nil {
		log.Println("Failed to encode filenames:", err)
	}
}

func handleContentFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Not a GET method", http.StatusMethodNotAllowed)
		return
	}

	filename, err := url.PathUnescape(strings.TrimPrefix(r.URL.Path, "/CCIC/content/"))
	if err != nil || filename == "" || filepath.Base(filename) != filename || strings.ToLower(filepath.Ext(filename)) != ".png" {
		http.Error(w, "Invalid content name", http.StatusBadRequest)
		return
	}

	http.ServeFile(w, r, filepath.Join(outputFilePath, filename))
}

func handleChangeContent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Not a POST method", http.StatusMethodNotAllowed)
		return
	}

	content, err := os.ReadDir(outputFilePath)
	if err != nil {
		http.Error(w, "failed to get avaliable content", http.StatusBadGateway)
		log.Println("Failed to os.ReadDir for content:", err)
		return
	}

	var filenames []string
	targetExt := ".png"

	for _, entry := range content {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) == targetExt {
			filenames = append(filenames, entry.Name())
		} else {
			continue
		}
	}

	var jsonContent CurrentContent

	jsonDecodeErr := json.NewDecoder(r.Body).Decode(&jsonContent)
	if jsonDecodeErr != nil {
		http.Error(w, "Failed to decode selected content json", http.StatusBadRequest)
		log.Println("Failed to decode selected content json:", jsonDecodeErr)
		return
	}

	found := false

	// iterate over the slice
	for _, v := range filenames {

		// check if the strings match
		if v == jsonContent.Name {
			found = true
			break
		}
	}

	if found {
		mtx.Lock()
		serverCurrentContent = jsonContent
		w.WriteHeader(http.StatusOK)
		mtx.Unlock()
	} else {
		http.Error(w, "Json content doesn't exist", http.StatusNotFound)
		log.Println("Json content doesn't exist")
	}
}

func StartServer(addr string, dist string) {
	outputFilePath = dist

	http.HandleFunc("/CCIC/uploadimage", handleImageUpload)
	http.HandleFunc("/CCIC/uploadgif", handleGifUpload)
	http.HandleFunc("/style.css", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/style.css")
	})
	http.HandleFunc("/main.js", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/main.js")
	})
	http.HandleFunc("/", handleStartPage)
	http.HandleFunc("/setSize", handleMonitorSizeSet)
	http.HandleFunc("/CCMonitorGetUpdateStatus", handleImageUpdate)
	http.HandleFunc("/CCInitContent", handleinitContent)
	http.HandleFunc("/getAvaliableContent", handleGetContentAvaliable)
	http.HandleFunc("/ChangeCurrentContent", handleChangeContent)
	http.HandleFunc("/CCIC/content/", handleContentFile)

	http.ListenAndServe(addr, nil)
}

func ForceContentInitialize() {
	contentInitialized = true
	serverCurrentContent.Name = "ImageInCaseOfContentMissing"
	serverCurrentContent.Anim = false
}

func DecodeGif(filePath string, outputDir string) error {
	inputFile, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open GIF: %w", err)
	}
	defer inputFile.Close()

	gifData, err := gif.DecodeAll(inputFile)
	if err != nil {
		return fmt.Errorf("decode GIF: %w", err)
	}

	if len(gifData.Image) == 0 {
		return fmt.Errorf("GIF contains no frames")
	}

	bounds := image.Rect(0, 0, gifData.Config.Width, gifData.Config.Height)
	canvas := image.NewRGBA(bounds)
	var previousCanvas *image.RGBA

	for i, frame := range gifData.Image {
		if i > 0 && i-1 < len(gifData.Disposal) {
			switch gifData.Disposal[i-1] {
			case gif.DisposalBackground:
				draw.Draw(canvas, gifData.Image[i-1].Bounds(), image.Transparent, image.Point{}, draw.Src)
			case gif.DisposalPrevious:
				if previousCanvas != nil {
					draw.Draw(canvas, bounds, previousCanvas, bounds.Min, draw.Src)
				}
			}
		}

		if i < len(gifData.Disposal) && gifData.Disposal[i] == gif.DisposalPrevious {
			previousCanvas = image.NewRGBA(bounds)
			draw.Draw(previousCanvas, bounds, canvas, bounds.Min, draw.Src)
		}

		draw.Draw(canvas, frame.Bounds(), frame, frame.Bounds().Min, draw.Over)

		outputPath := filepath.Join(outputDir, fmt.Sprintf("frame_%06d.png", i))
		outputFile, err := os.Create(outputPath)
		if err != nil {
			return fmt.Errorf("create frame %d: %w", i, err)
		}

		err = png.Encode(outputFile, canvas)
		closeErr := outputFile.Close()
		if err != nil {
			return fmt.Errorf("encode frame %d: %w", i, err)
		}
		if closeErr != nil {
			return fmt.Errorf("close frame %d: %w", i, closeErr)
		}
	}

	return nil
}
