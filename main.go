package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/gin-gonic/gin"
)

const (
	QUOTE_URL = "https://dummyjson.com/quotes/random"
	IMAGE_URL = "https://picsum.photos/800/600"
)

type DummyQuote struct {
	Quote  string `json:"quote"`
	Author string `json:"author"`
}

type InspirationResponse struct {
	QuoteInfo struct {
		Text   string `json:"text"`
		Author string `json:"author"`
	} `json:"quote_info"`
	ImageURL string `json:"image_url"`
}

func main() {
	r := gin.Default()

	apiV1 := r.Group("/api/v1")
	{
		apiV1.GET("/inspire-me", getInspirationHandler)
	}

	port := os.Getenv("PORT")

	if port == "" {
		port = "8080"
	}

	log.Printf("Starting server on port %s...", port)

	log.Println("Starting server on port", port)

	r.Run(":" + port)
}

func getInspirationHandler(c *gin.Context) {

	var wg sync.WaitGroup
	wg.Add(2)

	quoteChan := make(chan DummyQuote, 1)
	imageChan := make(chan string, 1)
	errChan := make(chan error, 2)

	go func() {
		defer wg.Done()

		resp, err := http.Get(QUOTE_URL)
		if err != nil {
			errChan <- err
			return
		}
		defer resp.Body.Close()

		var quoteData DummyQuote
		if err := json.NewDecoder(resp.Body).Decode(&quoteData); err != nil {
			errChan <- err
			return
		}

		quoteChan <- quoteData
	}()

	go func() {
		defer wg.Done()

		resp, err := http.Get(IMAGE_URL)
		if err != nil {
			errChan <- err
			return
		}
		defer resp.Body.Close()

		finalImageURL := resp.Request.URL.String()

		imageChan <- finalImageURL
	}()

	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	quote := <-quoteChan
	imageURL := <-imageChan

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

	c.JSON(http.StatusOK, response)
}
