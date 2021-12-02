package artistimage

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/gocolly/colly"
	"github.com/nfnt/resize"
)

const (
	userAgent   = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/74.0.3729.169 Safari/537.36"
	imageHeight = 150
)

func GetArtistImage(artistName string) ([]byte, error) {
	normalizedArtistName := strings.ToLower(url.QueryEscape(artistName))

	if cached := getCachedImage(normalizedArtistName); cached != nil {
		return cached, nil
	}

	imageWebPageURL, err := getImagesURLForArtist(normalizedArtistName)
	if err != nil {
		return nil, err
	}

	imageURL, err := getImageURLAtHTMLURL(imageWebPageURL)
	if err != nil {
		return nil, err
	}

	imageData, err := getImageDataAtURL(imageURL)
	if err != nil {
		return nil, err
	}

	resizedImage, err := resizeImage(imageData, imageHeight)
	if err != nil {
		return nil, err
	}

	setCachedImage(normalizedArtistName, resizedImage)

	return resizedImage, nil
}

func getImageDataAtURL(imageURL string) ([]byte, error) {
	resp, err := http.Get(imageURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	d, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return d, err
}

func getImagesURLForArtist(artist string) (string, error) {
	imagesLocation := ".header-new-gallery"

	u := "https://www.last.fm/music/" + artist
	c := colly.NewCollector()
	c.UserAgent = userAgent

	var imagesURL string
	var returnedError error

	c.OnHTML(imagesLocation, func(e *colly.HTMLElement) {
		u := "https://www.last.fm" + e.Attr("href")
		imagesURL = u
	})

	// Before making a request print "Visiting ..."
	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL.String())
	})

	if err := c.Visit(u); err != nil {
		log.Println(err)
	}

	return imagesURL, returnedError
}

func getImageURLAtHTMLURL(imageHtmlPageURL string) (string, error) {
	imageLocation := ".js-gallery-image"

	c := colly.NewCollector()
	c.UserAgent = userAgent

	var imageURL string

	c.OnHTML(imageLocation, func(e *colly.HTMLElement) {
		imageURL = e.Attr("src")
	})

	if err := c.Visit(imageHtmlPageURL); err != nil {
		log.Println(err)
	}

	return imageURL, nil
}

func resizeImage(imageData []byte, height uint) ([]byte, error) {
	originalImage, _, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return nil, err
	}

	newImage := resize.Resize(0, height, originalImage, resize.Lanczos3)

	// Encode uses a Writer, use a Buffer if you need the raw []byte
	var result bytes.Buffer
	err = jpeg.Encode(&result, newImage, nil)
	return result.Bytes(), err
}
