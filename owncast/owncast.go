package owncast

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path"
)

var owncastAccessToken string
var owncastServerHost string

// actionType represents an action you want to perform on Owncast.
type actionType = string

const (
	actionMessage  actionType = "/api/integrations/chat/action"
	systemMessage             = "/api/integrations/chat/system"
	setStreamTitle            = "/api/integrations/streamtitle"
)

// Message represents an Owncast chat-based message.
type Message struct {
	Author string `json:"author,omitempty"`
	Body   string `json:"body"`
}

// Config is a generic wrapper around an Owncast config value.
type Config struct {
	Value string `json:"value"`
}

// Setup will initialize the Owncast integration with a server url and access token.
func Setup(server string, accessToken string) {
	owncastServerHost = server
	owncastAccessToken = accessToken
}

// SendActionMessage will send an action message to Owncast chat.
func SendActionMessage(text string) error {
	jsonValue, _ := json.Marshal(Message{
		Body: text,
	})

	return send(actionMessage, jsonValue)
}

// SetStreamTitle will set the stream title.
func SetStreamTitle(title string) error {
	jsonValue, _ := json.Marshal(Config{
		Value: title,
	})

	return send(setStreamTitle, jsonValue)
}

func send(action actionType, data []byte) error {
	url, _ := url.Parse(owncastServerHost)
	url.Path = path.Join(url.Path, action)

	var bearer = "Bearer " + owncastAccessToken
	req, err := http.NewRequest("POST", url.String(), bytes.NewBuffer(data))
	req.Header.Add("Authorization", bearer)

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error with response: ", err)
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("Error with Owncast request:", err, body)
		return err
	}

	return nil
}
