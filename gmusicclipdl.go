package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"code.google.com/p/gopass"

	"github.com/amir/gpm"
	"github.com/atotto/clipboard"
)

func main() {
	pw, _ := gopass.GetPass("Enter password: ")

	client := NewGoogleMusic("bourget.alexandre@gmail.com", pw)
	log.Println("Logging in...")
	err := client.Login()
	if err != nil {
		log.Println("Error authenticating:", err)
		return
	}
	log.Println(".. done")

	log.Println("Listening for xclip...")
	initialWait := 0
	downloadList := make(chan string, 50)
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
			downloadList <- trackID
		}
	}
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

	filename := fmt.Sprintf("/home/abourget/Musique/%s - %s - %s.mp3", info.Title, info.Artist, info.Album)
	log.Println("Launching dl for", filename)

	log.Println("- getting link")
	mp3Url, err := gm.MP3StreamURL(info.Nid, "3e4e2ed2cd90976f")
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
	cmd := exec.Command("id3v2", filename, "--album", track.Album, "--artist", track.Artist, "--song", track.Title)
	log.Println("Writing ID3 tags")
	_, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("WriteID3 error: %s", err)
	}

	return nil
}
