package main

import "github.com/gabek/vadeo/owncast"

func setupOwncast() {
	if _config.OwncastAccessToken != "" && _config.OwncastServerURL != "" {
		owncast.Setup(_config.OwncastServerURL, _config.OwncastAccessToken)
	}
}
