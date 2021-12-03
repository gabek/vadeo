package main

import (
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gabek/vadeo/artistinfo"
	"github.com/gabek/vadeo/audio"
	"github.com/gabek/vadeo/metadata"
	"github.com/gabek/vadeo/owncast"
)

const (
	_pollStreamSeconds = time.Second * 9
)

var (
	_config = loadConfig()

	_artistTextFile  = filepath.Join(os.TempDir(), "vadeo-artist")
	_trackTextFile   = filepath.Join(os.TempDir(), "vadeo-track")
	_artistImageFile = filepath.Join(os.TempDir(), "vadeo-artist-image")

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

	audioBitrate := fmt.Sprintf("%dk", _config.AudioBitrate)

	pr, pw := io.Pipe()

	go streamAudio(pw)

	startingInput := "v"
	if !_config.UseAudioVisualizer {
		startingInput = "1:v"
	}
	filters := []string{
		// Overlay box
		fmt.Sprintf(`[%s]drawbox=y=ih-ih/4:color=black@0.5:width=iw:height=130:t=100[v]`, startingInput),

		// Artist & track text
		fmt.Sprintf(`[v]drawtext=fontsize=%d:fontcolor=White:fontfile="%s":textfile="%s":y=h-h/4+20:x=20:reload=1, drawtext=fontsize=%d:fontcolor=White:fontfile="%s":textfile="%s":y=h-h/4+80:x=20:reload=1, format=yuv420p[v]`, _config.ArtistFontSize, _config.FontFile, _artistTextFile, _config.TrackFontSize, _config.FontFile, _trackTextFile),

		// Logo
		`[v][2:v]overlay=x=(main_w-overlay_w-20):y=20[v]`,
	}
	if _config.UseArtistImage {
		// Artist image position.
		artistImage := `[v][3:v]overlay=x=(main_w-overlay_w-25):y=(main_h-main_h/4+10)[v]`
		filters = append(filters, artistImage)
	}
	if _config.UseAudioVisualizer {
		filters = append([]string{`[0:a]showwaves=mode=cline:s=hd720:colors=White@0.2|Blue@0.3|Black@0.3|Purple@0.3[v]; [1:v][v]overlay[v]`}, filters...)
	}

	filter := `-filter_complex "` + strings.Join(filters, "; ") + `"`

	var artistImageInput string
	if _config.UseArtistImage {
		artistImageInput = strings.Join([]string{
			"-stream_loop -1",
			"-re",
			"-f image2",
			"-i ", _artistImageFile,
		}, " ")
	}

	flags := []string{
		"ffmpeg",
		"-y",
		"-threads", "0",
		"-thread_queue_size", "9999",

		// MP3 stream
		"-re",
		"-f", "mp3", "-i", "pipe:0",

		// Video loop
		"-re",
		"-thread_queue_size", "9999",
		"-stream_loop", "-1",
		"-i background.mp4",

		// Logo
		"-stream_loop -1",
		"-re",
		"-f image2",
		"-i logo.png",

		// Artist image
		artistImageInput,

		// Visualization and overlays
		filter,
		"-map", "[v]",

		// Output
		"-map", "0:a:0",
		"-c:v", "libx264",
		"-preset", _config.CPUUsage,
		"-profile:v", "high",
		"-pix_fmt", "yuv420p",
		"-g", strconv.Itoa(_config.VideoFramerate * 2),
		"-crf", fmt.Sprintf("%d", _config.VideoQualityLevel),

		`-tune fastdecode`, `-tune zerolatency`,
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
		panic("Error streaming video.  Is your destination and key correct?  Check log.txt for troubleshooting: " + err.Error())
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

func updateOwncast(artist, track string) {
	if _config.OwncastAccessToken != "" {
		go func() {
			// A bit of a hack to offset the fact that the video stream
			// will be multiple seconds behind.
			time.Sleep(6 * time.Second)

			u := "https://www.last.fm/music/" + artist
			content := "Now Playing"
			hasImage := false
			if image, err := artistinfo.GetArtistImageURL(artist); err == nil && image != "" {
				hasImage = true
				content += fmt.Sprintf(`<a href="%s"><center><img src="%s" /></center></a>`, u, image)
			}

			if hasImage {
				content += fmt.Sprintf(`<a href="%s">`, u)
			}

			content += fmt.Sprintf(`<center><strong>%s - %s</strong></center>`, artist, track)
			if hasImage {
				content += `</a>`
			}

			if err := owncast.SendSystemMessage(content); err != nil {
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

	go updateOwncast(artist, track)

	if err := os.WriteFile(_artistTextFile, []byte(artist), 0644); err != nil {
		log.Println("unable to write artist text file:", err)
		return
	}

	if err := os.WriteFile(_trackTextFile, []byte(track), 0644); err != nil {
		log.Println("unable to write track text file:", err)
		return
	}

	if _config.UseArtistImage {
		imageData, err := artistinfo.GetArtistImage(artist)
		if err != nil {
			log.Println("unable to download artist image", err)
		}

		if err := os.WriteFile(_artistImageFile, imageData, 0644); err != nil {
			log.Println("unable to write artist image file:", err)
			return
		}
	}
}
