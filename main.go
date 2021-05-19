package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strings"
	"syscall"
	"time"

	"github.com/gabek/vadeo/owncast"
	shoutcast "github.com/gabek/vadeo/shoutcast"
)

const (
	_audioPipeFile  = "./.pipe.mp3"
	_artistTextFile = "./.artist.txt"
	_trackTextFile  = "./.track.txt"
)

var _config = loadConfig()
var _pipe *os.File

var _stationTitle = ""
var _stationDescription = ""

func main() {
	setup()
	go connectToStation()
	start()
}

func setup() {
	setupOwncast()

	if !doesFileExist(_audioPipeFile) {
		err := syscall.Mkfifo(_audioPipeFile, 0666)
		if err != nil {
			log.Fatalln(err)
		}
	}
}

func start() {
	rtmpDestination, err := url.Parse(_config.StreamingURL)
	if err != nil {
		panic(err)
	}
	rtmpDestination.Path = path.Join(rtmpDestination.Path, _config.StreamingKey)

	f, err := os.OpenFile(_audioPipeFile, os.O_RDWR, os.ModeNamedPipe)
	_pipe = f
	if err != nil {
		fmt.Println(err)
	}

	defer _pipe.Close()

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

	filter := fmt.Sprintf(`-filter_complex "[0:a]showwaves=mode=cline:s=hd720:colors=White@0.2|Blue@0.3|Black@0.3|Purple@0.3[v]; [1:v][v]overlay[v]; [v]drawbox=y=ih-ih/4:color=black@0.5:width=iw:height=130:t=100, drawtext=fontsize=40:fontcolor=White:fontfile=FreeSerif.ttf:textfile="%s"::y=h-h/4+20:x=20:reload=1, drawtext=fontsize=35:fontcolor=White:fontfile=FreeSerif.ttf:textfile="%s":y=h-h/4+80:x=20:reload=1, format=yuv420p[v]; [v]overlay=x=(main_w-overlay_w-20):y=20,format=rgba,colorchannelmixer=aa=0.5[v]; [v]setpts=PTS-STARTPTS[v]"`, _artistTextFile, _trackTextFile)
	flags := []string{
		"ffmpeg",
		"-y",

		"-thread_queue_size", "9999",
		"-re",
		"-f", "mp3", "-i", _audioPipeFile,

		"-re",
		// "-use_wallclock_as_timestamps", "1",
		"-thread_queue_size", "9999",
		"-stream_loop", "-1",
		"-r", fmt.Sprintf("%d", _config.VideoFramerate),
		// "-use_wallclock_as_timestamps", "1",
		"-i background.mp4",

		"-i logo.png",
		filter,
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
		"2> log.txt",
	}

	cmd := exec.Command("sh", "-c", strings.Join(flags, " "))

	if err != nil {
		panic(err)
	}

	err = cmd.Start()
	if err != nil {
		panic("Error starting ffmpeg.  Are you sure it's installed?")
	}
	err = cmd.Wait()
	if err != nil {
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

	// go func() {
	_, err = io.Copy(_pipe, stream)
	if err != nil {
		panic(fmt.Errorf("unable to write to %s: %s", _audioPipeFile, err))
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
