package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/url"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/gabek/vadeo/owncast"
	shoutcast "github.com/gabek/vadeo/shoutcast"
	mp3 "github.com/hajimehoshi/go-mp3"
)

const (
	_artistTextFile = "./.artist.txt"
	_trackTextFile  = "./.track.txt"
)

var _config = loadConfig()
var stdin io.WriteCloser

var _stationTitle = ""
var _stationDescription = ""

func main() {
	setup()
	go connectToStation()
	start()
}

func setup() {
	setupOwncast()
}

func start() {
	rtmpDestination, err := url.Parse(_config.StreamingURL)
	if err != nil {
		panic(err)
	}
	rtmpDestination.Path = path.Join(rtmpDestination.Path, _config.StreamingKey)

	if _config.AudioBitrate == 0 {
		_config.AudioBitrate = 128
	}

	if _config.VideoQualityLevel == 0 {
		_config.VideoQualityLevel = 25
	}

	if _config.CPUUsage == "" {
		_config.CPUUsage = "veryfast"
	}

	if _config.VideoFramerate == 0 {
		_config.VideoFramerate = 24
	}

	audioBitrate := fmt.Sprintf("%dk", _config.AudioBitrate)

	log.Printf("Vadeo is configured to send a %dfps video at a video quality crf of %d with %s audio to %s.", _config.VideoFramerate, _config.VideoQualityLevel, audioBitrate, _config.StreamingURL)

	filter := fmt.Sprintf(`[0:a]showwaves=mode=cline:s=hd720:colors=White@0.2|Blue@0.3|Black@0.3|Purple@0.3[v];[1:v][v]overlay[v];[v]drawbox=y=ih-ih/4:color=black@0.5:width=iw:height=130:t=100,drawtext=fontsize=40:fontcolor=White:fontfile=FreeSerif.ttf:textfile=%s::y=h-h/4+20:x=20:reload=1,drawtext=fontsize=35:fontcolor=White:fontfile=FreeSerif.ttf:textfile=%s:y=h-h/4+80:x=20:reload=1,format=yuv420p[v];[v]overlay=x=(main_w-overlay_w-20):y=20,format=rgba,colorchannelmixer=aa=0.5[v];[v]setpts=PTS-STARTPTS[v]`, _artistTextFile, _trackTextFile)
	flags := []string{
		"-y",

		"-thread_queue_size", "9999",
		"-re",
		"-f", "s32le", "-i", "pipe:", //_audioPipeFile,

		"-re",
		// "-use_wallclock_as_timestamps", "1",
		"-thread_queue_size", "9999",
		"-stream_loop", "-1",
		"-r", fmt.Sprintf("%d", _config.VideoFramerate),
		// "-use_wallclock_as_timestamps", "1",
		"-i", "background.mp4",

		"-i", "logo.png",
		"-filter_complex", filter,
		// filter,
		"-map", "[v]",
		"-map", "0:a:0",
		"-fflags", "+genpts",
		"-c:v", "libx264",
		"-preset", _config.CPUUsage,
		"-profile:v", "high",
		"-pix_fmt", "yuv420p",
		"-tune", "zerolatency",
		"-g", "30",
		"-crf", fmt.Sprintf("%d", _config.VideoQualityLevel),
		"-c:a", "aac", "-b:a", audioBitrate, "-ar", "44100",
		"-threads", "0",
		"-f", "flv",
		"-flvflags", "no_duration_filesize",
		rtmpDestination.String(),
		// "2> log.txt",
	}

	fmt.Println(strings.Join(flags, " "))
	cmd := exec.Command("/usr/local/bin/ffmpeg", flags...)
	stdin, err = cmd.StdinPipe()

	if err != nil {
		panic(err)
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		fmt.Println(fmt.Sprint(err) + ": " + stderr.String())

		panic(err)
		panic("Error starting ffmpeg.  Are you sure it's installed?")
	}
	err = cmd.Wait()
	if err != nil {
		panic(err)
		panic("Error streaming video.  Is your destination and key correct?  Check log.txt for troubleshooting.")
	}
}

func connectToStation() {
	stream, err := shoutcast.Open(_config.AudioURL)
	if err != nil {
		panic(fmt.Errorf("error connecting to %s %s", _config.AudioURL, err))
	}

	_stationTitle = stream.Name
	_stationDescription = stream.Description

	log.Println("Connected to", stream.Name, stream.Description)

	stream.MetadataCallbackFunc = stationMetadataChanged

	decoder, err := mp3.NewDecoder(stream)
	if err != nil {
		panic(err)
	}
	// go func() {
	if _, err = io.Copy(stdin, decoder); err != nil {
		panic(fmt.Errorf("unable to write audio: %s", err))
	}
	// }()
}

func stationMetadataChanged(m *shoutcast.Metadata) {
	log.Println("Now playing: ", m.StreamTitle)
	components := strings.SplitN(m.StreamTitle, " - ", 2)
	artist := ""
	track := ""

	artist = components[0]
	if len(components) > 1 {
		track = components[1]
	}

	if artist == "" && track == "" {
		artist = _stationTitle
		track = _stationDescription
	} else if track == "" {
		track = _stationTitle
	}

	ioutil.WriteFile(_artistTextFile, []byte(artist), 0644)
	ioutil.WriteFile(_trackTextFile, []byte(track), 0644)

	if _config.OwncastAccessToken != "" {
		go func() {
			// A bit of a hack to offset the fact that the video stream
			// will be multiple seconds behind.
			time.Sleep(10 * time.Second)
			owncast.SetStreamTitle("Now Playing: " + m.StreamTitle)
		}()
	}
}
