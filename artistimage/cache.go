package artistimage

import (
	"os"
	"path/filepath"
)

var cacheLocation = filepath.Join(os.TempDir(), "vadeo-artist-cache")

func setCachedImage(name string, image []byte) {
	f := filepath.Join(cacheLocation, name)
	os.WriteFile(f, image, 0644)
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
