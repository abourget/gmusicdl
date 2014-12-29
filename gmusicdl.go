package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/amir/gpm"
	"github.com/atotto/clipboard"
)

var config struct {
	Email     string
	Password  string
	OutputDir string `json:"output_dir"`
	DeviceID  string `json:"device_id"` // Valid Android Device ID, 16 chrs hex
}

func main() {
	if err := readConfig(); err != nil {
		log.Fatalln("Error reading config:", err)
	}

	client := NewGoogleMusic(config.Email, config.Password)
	log.Println("Logging in...")
	err := client.Login()
	if err != nil {
		log.Println("Error authenticating:", err)
		return
	}
	log.Println(".. done")

	log.Println("Listening for xclip...")
	initialWait := 0
	go client.manageDownloads()

	for {
		time.Sleep(time.Duration(initialWait) * time.Second)
		initialWait = 1
		clip, err := clipboard.ReadAll()
		if err != nil {
			log.Println("Error running clipboard progs", err)
			return
		}

		if strings.HasPrefix(clip, "https://play.google.com/music/m/") {
			trackID := clip[32:]
			if _, ok := client.seenList[trackID]; ok {
				continue
			}
			client.seenList[trackID] = true
			log.Println("Pushing track for download", trackID)
			client.trackIDch <- trackID
		}
	}
}

func readConfig() (err error) {
	cnt, err := ioutil.ReadFile("gmusicdl.conf")
	if err != nil {
		return
	}

	err = json.Unmarshal(cnt, &config)
	if err != nil {
		return
	}

	return nil
}

type GoogleMusic struct {
	*gpm.Client
	trackIDch chan string
	seenList  map[string]bool
}

func NewGoogleMusic(login, pass string) *GoogleMusic {
	return &GoogleMusic{
		Client:    gpm.New(login, pass),
		trackIDch: make(chan string, 50),
		seenList:  make(map[string]bool),
	}
}

func (gm *GoogleMusic) manageDownloads() {
	for {
		log.Println("Waiting for next track...")
		trackID := <-gm.trackIDch
		info := gm.fetchTrackInfo(trackID)
		if info == nil {
			continue
		}

		filename, err := gm.launchDownload(info)
		if err != nil {
			log.Println("Download error:", err)
			continue
		}

		err = gm.writeID3(info, filename)
		if err != nil {
			log.Println(err)
		}

	}
}

func (gm *GoogleMusic) fetchTrackInfo(trackID string) *gpm.Track {
	log.Println("Getting track info...")

	info, err := gm.TrackInfo(trackID)
	if err != nil {
		log.Println("Couldn't get TrackInfo:", err)
		return nil
	}

	log.Printf("  got it: %#v", info)
	return &info
}
func (gm *GoogleMusic) launchDownload(info *gpm.Track) (string, error) {
	filename := path.Join(config.OutputDir, fmt.Sprintf("%s - %s - %s.mp3", info.Title, info.Artist, info.Album))
	log.Println("Launching dl for", filename)

	log.Println("- getting link")
	mp3Url, err := gm.MP3StreamURL(info.Nid, config.DeviceID)
	if err != nil {
		return "", fmt.Errorf("MP3StreamURL error: %s", err)
	}

	log.Println("- downloading song")
	resp, err := http.Get(mp3Url)
	if err != nil {
		return "", fmt.Errorf("Couldn't download song: %s", err)
	}

	out, err := os.Create(filename)
	if err != nil {
		return "", fmt.Errorf("Couldn't write output mp3: %s", err)

	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", fmt.Errorf("Couldn't read or write MP3: %s", err)
	}

	log.Println("Done")

	return filename, nil
}

func (gm *GoogleMusic) writeID3(track *gpm.Track, filename string) error {
	args := []string{
		filename,
		"--album", track.Album,
		"--artist", track.Artist,
		"--song", track.Title,
	}
	if track.Year != 0 {
		args = append(args, "--year", fmt.Sprintf("%d", track.Year))
	}
	if track.TrackNumber != 0 {
		args = append(args, "--track", fmt.Sprintf("%02d", track.TrackNumber))
	}

	cmd := exec.Command("id3v2", args...)
	log.Println("Writing ID3 tags")
	_, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("WriteID3 error: %s", err)
	}

	return nil
}
