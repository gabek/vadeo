package shoutcast

import (
	"log"
	"strings"
)

// Metadata represents the stream metadata sent by the server
type Metadata struct {
	StreamTitle string
}

// NewMetadata returns parsed metadata
func NewMetadata(b []byte) *Metadata {
	m := &Metadata{}

	props := strings.Split(string(b), ";")
	log.Print("[DEBUG] Received metadata: ", props)

	for _, prop := range props {
		if prop == "" {
			continue
		}
		parts := strings.Split(prop, "=")
		if parts[0] == "StreamTitle" {
			m.StreamTitle = strings.Trim(parts[1], "'")
		}
	}

	return m
}

// Equals compares two Metadata structures for equality
func (m *Metadata) Equals(other *Metadata) bool {
	if other == nil {
		return false
	}
	if m.StreamTitle != other.StreamTitle {
		return false
	}
	return true
}
