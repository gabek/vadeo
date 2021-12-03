package artistimage

import (
	"log"
	"os"
	"path/filepath"
)

var cacheLocation = filepath.Join(os.TempDir(), "vadeo-artist-cache")

func setCachedImage(name string, image []byte) {
	if !doesFileExist(cacheLocation) {
		if err := os.Mkdir(cacheLocation, 0770); err != nil {
			log.Println("error creating cache directory:", err)
			return
		}
	}

	f := filepath.Join(cacheLocation, name)
	if err := os.WriteFile(f, image, 0644); err != nil {
		log.Println("error writing cache file:", err)
	}
}

func getCachedImage(name string) []byte {
	f := filepath.Join(cacheLocation, name)
	if !doesFileExist(f) {
		return nil
	}

	b, _ := os.ReadFile(f)
	return b
}

func doesFileExist(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}

	return true
}
