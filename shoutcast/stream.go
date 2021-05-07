package shoutcast

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"time"
)

// MetadataCallbackFunc is the type of the function called when the stream metadata changes
type MetadataCallbackFunc func(m *Metadata)

type BytesCallbackFunc func(b []byte)

// Stream represents an open shoutcast stream.
type Stream struct {
	// The name of the server
	Name string

	// What category the server falls under
	Genre string

	// The description of the stream
	Description string

	// Homepage of the server
	URL string

	// Bitrate of the server
	Bitrate int

	// Optional function to be executed when stream metadata changes
	MetadataCallbackFunc MetadataCallbackFunc

	// Amount of bytes to read before expecting a metadata block
	metaint int

	// Stream metadata
	metadata *Metadata

	// The number of bytes read since last metadata block
	pos int

	// The underlying data stream
	rc io.ReadCloser
}

// Open establishes a connection to a remote server.
func Open(url string) (*Stream, error) {
	req, err := http.NewRequest("GET", url, nil)
	req.Header.Add("accept", "*/*")
	req.Header.Add("user-agent", "iTunes/12.9.2 (Macintosh; OS X 10.14.3) AppleWebKit/606.4.5")
	req.Header.Add("icy-metadata", "1")

	// Timeout for establishing the connection.
	// We don't want for the stream to timeout while we're reading it, but
	// we do want a timeout for establishing the connection to the server.
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	transport := &http.Transport{Dial: dialer.Dial}
	client := &http.Client{Transport: transport}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	// for k, v := range resp.Header {
	// 	log.Print("[DEBUG] HTTP header ", k, ": ", v[0])
	// }

	var bitrate int
	if rawBitrate := resp.Header.Get("icy-br"); rawBitrate != "" {
		bitrate, err = strconv.Atoi(rawBitrate)
		if err != nil {
			return nil, fmt.Errorf("cannot parse bitrate: %v", err)
		}
	}

	metaint, err := strconv.Atoi(resp.Header.Get("icy-metaint"))
	if err != nil {
		return nil, fmt.Errorf("cannot parse metaint: %v", err)
	}

	s := &Stream{
		Name:        resp.Header.Get("icy-name"),
		Genre:       resp.Header.Get("icy-genre"),
		Description: resp.Header.Get("icy-description"),
		URL:         resp.Header.Get("icy-url"),
		Bitrate:     bitrate,
		metaint:     metaint,
		metadata:    nil,
		pos:         0,
		rc:          resp.Body,
	}

	return s, nil
}

// Read implements the standard Read interface
func (s *Stream) Read(buf []byte) (dataLen int, err error) {
	dataLen, err = s.rc.Read(buf)

	checkedDataLen := 0
	uncheckedDataLen := dataLen
	for s.pos+uncheckedDataLen > s.metaint {
		offset := s.metaint - s.pos
		skip, e := s.extractMetadata(buf[checkedDataLen+offset:])
		if e != nil {
			err = e
			fmt.Println(err)
		}
		s.pos = 0
		if offset+skip > uncheckedDataLen {
			dataLen = checkedDataLen + offset
			uncheckedDataLen = 0
		} else {
			checkedDataLen += offset
			dataLen -= skip
			uncheckedDataLen = dataLen - checkedDataLen
			copy(buf[checkedDataLen:], buf[checkedDataLen+skip:])
		}
	}
	s.pos = s.pos + uncheckedDataLen

	return
}

// Close closes the stream
func (s *Stream) Close() error {
	return s.rc.Close()
}

func (s *Stream) extractMetadata(p []byte) (int, error) {
	var metabuf []byte
	var err error
	length := int(p[0]) * 16
	end := length + 1
	complete := false
	if length > 0 {
		if len(p) < end {
			// The provided buffer was not large enough for the metadata block to fit in.
			// Read whole metadata into our own buffer.
			metabuf = make([]byte, length)
			copy(metabuf, p[1:])
			n := len(p) - 1
			for n < length && err == nil {
				var nn int
				nn, err = s.rc.Read(metabuf[n:])
				n += nn
			}
			if n == length {
				complete = true
			} else if err == nil || err == io.EOF {
				err = io.ErrUnexpectedEOF
			}
		} else {
			metabuf = p[1:end]
			complete = true
		}
	}
	if complete {
		if m := NewMetadata(metabuf); !m.Equals(s.metadata) {
			s.metadata = m
			if s.MetadataCallbackFunc != nil {
				s.MetadataCallbackFunc(s.metadata)
			}
		}
	}
	return length + 1, err
}
