package metadata

import (
	"bufio"
	"bytes"
	"fmt"
	"net/http"
	"strconv"
)

// NowPlaying represents an open shoutcast stream.

// GetStreamTitle get the current song/show in an Icecast stream
func GetStreamTitle(streamUrl string) (string, error) {
	m, err := getStreamMetas(streamUrl)
	if err != nil {
		return "", err
	}
	// Should be at least "StreamTitle=' '"
	if len(m) < 15 {
		return "", nil
	}
	// Split meta by ';', trim it and search for StreamTitle
	for _, m := range bytes.Split(m, []byte(";")) {
		m = bytes.Trim(m, " \t")
		if !bytes.Equal(m[0:13], []byte("StreamTitle='")) {
			continue
		}
		return string(m[13 : len(m)-1]), nil
	}
	return "", nil
}

// get stream metadatas
func getStreamMetas(streamUrl string) ([]byte, error) {
	client := &http.Client{}
	req, _ := http.NewRequest("GET", streamUrl, nil)
	req.Header.Set("Icy-MetaData", "1")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	// We sent "Icy-MetaData", we should have a "icy-metaint" in return
	ih := resp.Header.Get("icy-metaint")
	if ih == "" {
		return nil, fmt.Errorf("no metadata")
	}
	// "icy-metaint" is how often (in bytes) should we receive the meta
	ib, err := strconv.Atoi(ih)
	if err != nil {
		return nil, err
	}

	reader := bufio.NewReader(resp.Body)

	// skip the first mp3 frame
	c, err := reader.Discard(ib)
	if err != nil {
		return nil, err
	}
	// If we didn't received ib bytes, the stream is ended
	if c != ib {
		return nil, fmt.Errorf("stream ended prematurally")
	}

	// get the size byte, that is the metadata length in bytes / 16
	sb, err := reader.ReadByte()
	if err != nil {
		return nil, err
	}
	ms := int(sb * 16)

	// read the ms first bytes it will contain metadata
	m, err := reader.Peek(ms)
	if err != nil {
		return nil, err
	}
	return m, nil
}

// Open establishes a connection to a remote server.
// func Open(url string) (*Stream, error) {
// 	req, err := http.NewRequest("GET", url, nil)
// 	req.Header.Add("accept", "*/*")
// 	req.Header.Add("user-agent", "iTunes/12.9.2 (Macintosh; OS X 10.14.3) AppleWebKit/606.4.5")
// 	req.Header.Add("icy-metadata", "1")

// 	// Timeout for establishing the connection.
// 	// We don't want for the stream to timeout while we're reading it, but
// 	// we do want a timeout for establishing the connection to the server.
// 	dialer := &net.Dialer{Timeout: 5 * time.Second}
// 	transport := &http.Transport{Dial: dialer.Dial}
// 	client := &http.Client{Transport: transport}
// 	resp, err := client.Do(req)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// for k, v := range resp.Header {
// 	// 	log.Print("[DEBUG] HTTP header ", k, ": ", v[0])
// 	// }

// 	var bitrate int
// 	if rawBitrate := resp.Header.Get("icy-br"); rawBitrate != "" {
// 		bitrate, err = strconv.Atoi(rawBitrate)
// 		if err != nil {
// 			return nil, fmt.Errorf("cannot parse bitrate: %v", err)
// 		}
// 	}

// 	metaint, err := strconv.Atoi(resp.Header.Get("icy-metaint"))
// 	if err != nil {
// 		return nil, fmt.Errorf("cannot parse metaint: %v", err)
// 	}

// 	s := &Stream{
// 		Name:        resp.Header.Get("icy-name"),
// 		Genre:       resp.Header.Get("icy-genre"),
// 		Description: resp.Header.Get("icy-description"),
// 		URL:         resp.Header.Get("icy-url"),
// 		Bitrate:     bitrate,
// 		metaint:     metaint,
// 		metadata:    nil,
// 		pos:         0,
// 		rc:          resp.Body,
// 	}

// 	return s, nil
// }

// // Read implements the standard Read interface
// func (s *Stream) Read(buf []byte) (dataLen int, err error) {
// 	dataLen, err = s.rc.Read(buf)

// 	checkedDataLen := 0
// 	uncheckedDataLen := dataLen
// 	for s.pos+uncheckedDataLen > s.metaint {
// 		offset := s.metaint - s.pos
// 		skip, e := s.extractMetadata(buf[checkedDataLen+offset:])
// 		if e != nil {
// 			err = e
// 			fmt.Println(err)
// 		}
// 		s.pos = 0
// 		if offset+skip > uncheckedDataLen {
// 			dataLen = checkedDataLen + offset
// 			uncheckedDataLen = 0
// 		} else {
// 			checkedDataLen += offset
// 			dataLen -= skip
// 			uncheckedDataLen = dataLen - checkedDataLen
// 			copy(buf[checkedDataLen:], buf[checkedDataLen+skip:])
// 		}
// 	}
// 	s.pos = s.pos + uncheckedDataLen

// 	return
// }

// // Close closes the stream
// func (s *Stream) Close() error {
// 	return s.rc.Close()
// }

// func (s *Stream) extractMetadata(p []byte) (int, error) {
// 	var metabuf []byte
// 	var err error
// 	length := int(p[0]) * 16
// 	end := length + 1
// 	complete := false
// 	if length > 0 {
// 		if len(p) < end {
// 			// The provided buffer was not large enough for the metadata block to fit in.
// 			// Read whole metadata into our own buffer.
// 			metabuf = make([]byte, length)
// 			copy(metabuf, p[1:])
// 			n := len(p) - 1
// 			for n < length && err == nil {
// 				var nn int
// 				nn, err = s.rc.Read(metabuf[n:])
// 				n += nn
// 			}
// 			if n == length {
// 				complete = true
// 			} else if err == nil || err == io.EOF {
// 				err = io.ErrUnexpectedEOF
// 			}
// 		} else {
// 			metabuf = p[1:end]
// 			complete = true
// 		}
// 	}
// 	if complete {
// 		if m := NewMetadata(metabuf); !m.Equals(s.metadata) {
// 			s.metadata = m
// 			if s.MetadataCallbackFunc != nil {
// 				s.MetadataCallbackFunc(s.metadata)
// 			}
// 		}
// 	}
// 	return length + 1, err
// }
