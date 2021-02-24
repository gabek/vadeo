package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/gabek/vadeo/owncast"
	shoutcast "github.com/gabek/vadeo/shoutcast"
)

const (
	audioPipe  = ".pipe.mp3"
	artistFile = ".artist.txt"
	trackFile  = ".track.txt"

	rtmpDestination = "rtmp://127.0.0.1/live/abc123"
	audioURL        = "http://14833.live.streamtheworld.com/SAM04AAC147.mp3"
)

var config = loadConfig()

var _pipe *os.File

func main() {
	if config.OwncastAccessToken != "" {
		owncastRootURL, err := url.Parse(config.StreamingURL)
		if err != nil {
			log.Println("Error parsing Owncast host: ", config.StreamingKey)
			return
		}

		owncastRootURL.Path = ""
		owncast.Setup(owncastRootURL.String(), config.OwncastAccessToken)
	}

	if !DoesFileExists(audioPipe) {
		err := syscall.Mkfifo(audioPipe, 0666)
		if err != nil {
			log.Fatalln(err)
		}
	}

	f, err := os.OpenFile(audioPipe, os.O_RDWR, os.ModeNamedPipe)
	_pipe = f
	if err != nil {
		fmt.Println(err)
	}

	defer _pipe.Close()

	filter := fmt.Sprintf(`-filter_complex "[0:a]showwaves=mode=cline:s=hd720:colors=White@0.2|Blue@0.3|Black@0.3|Purple@0.3[v]; [1:v][v]overlay[v]; [v]drawbox=y=ih-ih/4:color=black@0.5:width=iw:height=130:t=100, drawtext=fontsize=40:fontcolor=White:fontfile=FreeSerif.ttf:textfile="%s"::y=h-h/4+20:x=20:reload=1, drawtext=fontsize=35:fontcolor=White:fontfile=FreeSerif.ttf:textfile="%s":y=h-h/4+80:x=20:reload=1, format=yuv420p[v]; [v]overlay=x=(main_w-overlay_w-20):y=20,format=rgba,colorchannelmixer=aa=0.5[v]"`, artistFile, trackFile)
	flags := []string{
		"/usr/local/bin/ffmpeg",
		"-y",

		"-thread_queue_size", "9999",
		"-re", "-f", "mp3", "-i", audioPipe,

		"-thread_queue_size", "9999",
		"-stream_loop", "-1",

		"-i background.mp4",
		"-i logo.png",
		filter,
		"-map", "[v]",
		"-map", "0:a:0",
		"-c:v", "libx264", "-preset", "faster", "-b:v", "4000k",
		"-g", "24", "-c:a", "aac", "-b:a", "128k", "-ar", "44100",
		"-f", "flv",
		rtmpDestination,
		// "rtmp://173.255.213.200/live/abc123",
		"2> log.txt",
	}

	cmd := exec.Command("sh", "-c", strings.Join(flags, " "))

	if err != nil {
		panic(err)
	}

	stream, err := shoutcast.Open(audioURL)
	if err != nil {
		panic(err)
	}

	log.Println("Connected to", stream.Name, stream.Description)

	stream.MetadataCallbackFunc = func(m *shoutcast.Metadata) {
		log.Println("Now playing: ", m.StreamTitle)
		components := strings.Split(m.StreamTitle, " - ")
		artist := ""
		track := ""

		artist = components[0]
		if len(components) > 1 {
			track = components[1]
		}

		ioutil.WriteFile(artistFile, []byte(artist), 0644)
		ioutil.WriteFile(trackFile, []byte(track), 0644)

		if config.OwncastAccessToken != "" {
			owncast.SetStreamTitle(m.StreamTitle)
		}
	}

	go func() {
		_, err = io.Copy(_pipe, stream)
		if err != nil {
			panic(err)
		}
	}()

	err = cmd.Start()
	if err != nil {
		panic("Error starting ffmpeg.  Are you sure it's installed?")
	}
	err = cmd.Wait()
	if err != nil {
		panic("Error streaming video.  Is your destination and key correct?  Check log.txt for troubleshooting.")
	}

	for {
	}
}

// DoesFileExists checks if the file exists.
func DoesFileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}

	return true
}
