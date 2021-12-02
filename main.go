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
	"path/filepath"
	"strings"
	"time"

	"github.com/gabek/vadeo/artistimage"
	"github.com/gabek/vadeo/audio"
	"github.com/gabek/vadeo/metadata"
	"github.com/gabek/vadeo/owncast"
)

const (
	_pollStreamSeconds = time.Second * 7
)

var (
	_config = loadConfig()

	_artistTextFile  = filepath.Join(os.TempDir(), "vadio-artist")
	_trackTextFile   = filepath.Join(os.TempDir(), "vadio-track")
	_artistImageFile = filepath.Join(os.TempDir(), "vadio-artist-image")

	_stationDescription = ""
	_stationTitle       = ""
	_currentNowPlaying  = ""
)

func main() {
	setup()
	connectToStation()
	start()
}

func setup() {
	setupOwncast()
}

func streamAudio(pw *io.PipeWriter) {
	stream, err := audio.Connect(_config.AudioURL)
	if err != nil {
		panic(err)
	}
	if _, err := io.Copy(pw, stream.Stream); err != nil {
		panic(err)
	}

	_stationTitle = stream.Name
	_stationDescription = stream.Description

	log.Println("Streaming audio from", stream.Name)
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

	pr, pw := io.Pipe()

	go streamAudio(pw)

	filters := []string{
		// Audio visualizer
		`[0:a]showwaves=mode=cline:s=hd720:colors=White@0.2|Blue@0.3|Black@0.3|Purple@0.3[v]`,
		`[1:v][v]overlay[v]`,

		// Overlay box + text
		fmt.Sprintf(`[v]drawbox=y=ih-ih/4:color=black@0.5:width=iw:height=130:t=100, drawtext=fontsize=40:fontcolor=White:fontfile=FreeSerif.ttf:textfile="%s"::y=h-h/4+20:x=20:reload=1, drawtext=fontsize=35:fontcolor=White:fontfile=FreeSerif.ttf:textfile="%s":y=h-h/4+80:x=20:reload=1, format=yuv420p[v]`, _artistTextFile, _trackTextFile),

		// Logo
		`[2:v]format=rgba,colorchannelmixer=aa=0.9[logo]`,
		`[v][logo]overlay=x=(main_w-overlay_w-20):y=20[v]`,

		// Artist image is scaled down and made into a square.
		`[3:v]scale=110:110,setsar=1:1,crop=110:110[artistimage]`,
		`[v][artistimage]overlay=x=(main_w-overlay_w-25):y=(main_h-main_h/4+10)[v]`,
	}
	filter := `-filter_complex "` + strings.Join(filters, "; ") + `"`

	flags := []string{
		"ffmpeg",
		"-y",

		"-thread_queue_size", "9999",

		// MP3 stream
		"-re",
		"-f", "mp3", "-i", "pipe:0",

		// Video loop
		"-re",
		"-thread_queue_size", "9999",
		"-stream_loop", "-1",
		"-r", fmt.Sprintf("%d", _config.VideoFramerate),
		"-i background.mp4",

		// Logo
		"-stream_loop -1",
		"-re",
		"-f image2",
		"-i logo.png",

		// Artist image
		"-stream_loop -1",
		"-re",
		"-f image2",
		"-i ", _artistImageFile,

		// Visualization and overlays
		filter,
		"-map", "[v]",

		// Output
		"-map", "0:a:0",
		"-c:v", "libx264",
		"-preset", _config.CPUUsage,
		"-profile:v", "high",
		"-pix_fmt", "yuv420p",
		"-g", "30",
		"-crf", fmt.Sprintf("%d", _config.VideoQualityLevel),
		"-c:a", "aac", "-b:a", audioBitrate, "-ar", "44100",

		"-f", "flv",
		rtmpDestination.String(),
		"2> log.txt",
	}

	log.Printf("Vadeo is configured to send a %dfps video at a video quality crf of %d with %s audio to %s.", _config.VideoFramerate, _config.VideoQualityLevel, audioBitrate, _config.StreamingURL)

	cmd := exec.Command("sh", "-c", strings.Join(flags, " "))
	cmd.Stdin = pr

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
	fetchNowPlaying()

	statsSaveTimer := time.NewTicker(_pollStreamSeconds)
	go func() {
		for range statsSaveTimer.C {
			fetchNowPlaying()
		}
	}()
}

func fetchNowPlaying() {
	nowPlaying, err := metadata.GetStreamTitle(_config.AudioURL)
	if err != nil {
		log.Println(err)

		// Use station name as a fallback.
		nowPlaying = _stationTitle + " - " + _stationDescription
	}

	if _currentNowPlaying != nowPlaying {
		stationMetadataChanged(nowPlaying)
	}
}

func updateOwncast(nowPlaying string) {
	if _config.OwncastAccessToken != "" {
		go func() {
			// A bit of a hack to offset the fact that the video stream
			// will be multiple seconds behind.
			time.Sleep(8 * time.Second)
			if err := owncast.SetStreamTitle(nowPlaying); err != nil {
				log.Println(err)
			}
		}()
	}
}

func stationMetadataChanged(nowPlaying string) {
	log.Println("Now playing: ", nowPlaying)

	os.Remove(_artistImageFile)

	_currentNowPlaying = nowPlaying
	components := strings.SplitN(nowPlaying, " - ", 2)
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

	go updateOwncast(nowPlaying)

	if err := ioutil.WriteFile(_artistTextFile, []byte(artist), 0644); err != nil {
		log.Println("unable to write artist text file:", err)
		return
	}

	if err := ioutil.WriteFile(_trackTextFile, []byte(track), 0644); err != nil {
		log.Println("unable to write track text file:", err)
		return
	}

	imageData, err := artistimage.GetArtistImage(artist)
	if err != nil {
		log.Println("unable to download artist image", err)
	}

	if err := ioutil.WriteFile(_artistImageFile, imageData, 0644); err != nil {
		log.Println("unable to write artist image file:", err)
		return
	}
}
