package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync" // Kita butuh ini untuk konkurensi (WaitGroups)

	"github.com/gin-gonic/gin"
)

// Alamat API eksternal
const (
	QUOTE_URL = "https://dummyjson.com/quotes/random"
	IMAGE_URL = "https://picsum.photos/800/600"
)

// --- Definisikan Struktur Data ---

// 1. Struktur untuk menampung respons dari dummyjson.com
// Kita hanya butuh 'quote' dan 'author', jadi kita abaikan sisanya.
type DummyQuote struct {
	Quote  string `json:"quote"`
	Author string `json:"author"`
}

// 2. Struktur untuk respons akhir API kita
type InspirationResponse struct {
	QuoteInfo struct {
		Text   string `json:"text"`
		Author string `json:"author"`
	} `json:"quote_info"`
	ImageURL string `json:"image_url"`
}

// --- Fungsi Utama ---

func main() {
	// 1. Inisialisasi Gin router
	r := gin.Default()

	// 2. Definisikan endpoint API kita
	// Kita akan mengelompokkannya di /api/v1
	apiV1 := r.Group("/api/v1")
	{
		apiV1.GET("/inspire-me", getInspirationHandler)
	}

	// 3. Jalankan server di port 8080
	log.Println("Starting server on port 8080...")
	r.Run(":8080") // Listen and serve on 0.0.0.0:8080
}

// --- Handler (Logika Inti) ---

// getInspirationHandler adalah fungsi yang menangani request
func getInspirationHandler(c *gin.Context) {
	// 💡 INI BAGIAN PENTING (KONKURENSI)
	// Kita akan menggunakan WaitGroup dan Channels untuk menunggu
	// kedua panggilan API selesai secara bersamaan.

	var wg sync.WaitGroup // Membuat "penghitung" goroutine
	wg.Add(2) // Kita akan menjalankan 2 tugas: 1. Ambil Quote, 2. Ambil Image

	// Channels adalah "pipa" untuk mengirim data antar goroutine
	quoteChan := make(chan DummyQuote, 1)
	imageChan := make(chan string, 1)
	errChan := make(chan error, 2) // Channel untuk menampung error

	// --- Tugas 1: Ambil Kutipan (Goroutine) ---
	go func() {
		defer wg.Done() // Tandai tugas ini selesai saat fungsi berakhir
		
		resp, err := http.Get(QUOTE_URL)
		if err != nil {
			errChan <- err
			return
		}
		defer resp.Body.Close()

		var quoteData DummyQuote
		// Decode JSON langsung dari respons body
		if err := json.NewDecoder(resp.Body).Decode(&quoteData); err != nil {
			errChan <- err
			return
		}
		
		// Kirim data kutipan ke channel
		quoteChan <- quoteData
	}()

	// --- Tugas 2: Ambil URL Gambar (Goroutine) ---
	go func() {
		defer wg.Done() // Tandai tugas ini selesai

		// http.Get otomatis mengikuti redirect
		resp, err := http.Get(IMAGE_URL)
		if err != nil {
			errChan <- err
			return
		}
		defer resp.Body.Close()

		// Trik Picsum: URL final ada di objek Request setelah redirect
		finalImageURL := resp.Request.URL.String()
		
		// Kirim URL gambar ke channel
		imageChan <- finalImageURL
	}()

	// --- Tunggu & Gabungkan Hasil ---
	wg.Wait()      // Blokir/tunggu sampai kedua tugas memanggil wg.Done()
	close(errChan) // Tutup channel error

	// Cek apakah ada error
	for err := range errChan {
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	// Ambil hasil dari channel
	quote := <-quoteChan
	imageURL := <-imageChan

	// Buat respons akhir kita
	response := InspirationResponse{
		QuoteInfo: struct {
			Text   string `json:"text"`
			Author string `json:"author"`
		}{
			Text:   quote.Quote,
			Author: quote.Author,
		},
		ImageURL: imageURL,
		
	}

	// Kirim respons JSON ke client
	c.JSON(http.StatusOK, response)
}