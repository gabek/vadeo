package artistinfo

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/jpeg"
	"image/png"
	_ "image/png"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/gocolly/colly"
)

const (
	userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/74.0.3729.169 Safari/537.36"
)

func GetArtistImageURL(artistName string) (string, error) {
	normalizedArtistName := strings.ToLower(url.QueryEscape(artistName))
	imageWebPageURL, err := getImagesURLForArtist(normalizedArtistName)
	if err != nil {
		return "", err
	}

	imageURL, err := getImageURLAtHTMLURL(imageWebPageURL)
	if err != nil {
		return "", err
	}
	return imageURL, nil
}

func GetArtistImage(artistName string) ([]byte, error) {
	normalizedArtistName := strings.ToLower(url.QueryEscape(artistName))

	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	var imageDataBuffer bytes.Buffer
	if err := png.Encode(&imageDataBuffer, img); err != nil {
		log.Println(err)
	}
	defaultImageData := imageDataBuffer.Bytes()

	if cached := getCachedImage(normalizedArtistName); cached != nil {
		return cached, nil
	}

	imageWebPageURL, err := getImagesURLForArtist(normalizedArtistName)
	if err != nil {
		return defaultImageData, err
	}

	imageURL, err := getImageURLAtHTMLURL(imageWebPageURL)
	if err != nil {
		return defaultImageData, err
	}

	imageData, err := getImageDataAtURL(imageURL)
	if err != nil {
		return defaultImageData, err
	}

	setCachedImage(normalizedArtistName, imageData)

	return imageData, nil
}

func getImageDataAtURL(imageURL string) ([]byte, error) {
	resp, err := http.Get(imageURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	fullSizeImage, _, err := image.Decode(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("unable to decode artist image:%s ", err)
	}

	resizedImage := imaging.Resize(fullSizeImage, 0, 110, imaging.Lanczos)

	out := bytes.Buffer{}
	err = jpeg.Encode(&out, resizedImage, nil)

	return out.Bytes(), err
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
