package audio

import (
	"io"
	"net/http"
)

// AudioStream represents an internet radio station with a stream of audio.
type AudioStream struct {
	Stream      io.Reader
	Name        string
	Genre       string
	URL         string
	Description string
}

// Connect will connect to an audio stream and return a handle to the stream
// and additional metadata.
func Connect(url string) (AudioStream, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return AudioStream{}, err
	}

	req.Header.Add("accept", "*/*")
	req.Header.Add("user-agent", "vadeo")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return AudioStream{}, err
	}

	stream := AudioStream{
		Name:        resp.Header.Get("icy-name"),
		Genre:       resp.Header.Get("icy-genre"),
		Description: resp.Header.Get("icy-description"),
		URL:         resp.Header.Get("icy-url"),
		Stream:      resp.Body,
	}
	return stream, err
}
