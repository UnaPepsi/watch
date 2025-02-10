package main

import (
	"bytes"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

func main() {

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		file, err := os.ReadFile("index.html")

		w.Header().Set("Content-Type", "text/html")
		if err != nil {
			fmt.Fprint(w, "An error ocurred, please try again. Traceback:", err)
			return
		}
		entries, err := os.ReadDir("./series")
		if err != nil {
			fmt.Fprint(w, "An error ocurred, please try again. Traceback:", err)
			return
		}
		var seriesB strings.Builder
		for _, entry := range entries {
			if entry.IsDir() {
				contents, err := os.ReadDir("./series/" + entry.Name())
				if err != nil {
					fmt.Fprint(w, "An error ocurred, please try again. Traceback:", err)
					return
				}
				if len(contents) != 0 {
					seriesB.WriteString("<li><a href=\"/series/" + entry.Name() + "\">" + entry.Name() + "/</a></li>\n")
				}
			}
		}
		html := strings.Replace(string(file), "--add here", seriesB.String(), 1)
		fmt.Fprint(w, html)
	})

	//why the heck is the singular of series, also series
	mux.HandleFunc("/series/{series}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		path := r.PathValue("series")
		entries, err := os.ReadDir("./series/" + path)
		if err != nil {
			fmt.Fprint(w, "An error occured, please try again. Traceback:", err)
			return
		}
		var chaptersB strings.Builder
		for index, entry := range entries {
			if _, err := chaptersB.WriteString(fmt.Sprintf("<li><a href=/series/%v/chapter-%v>%v</a></li>\n", path, index+1, entry.Name())); err != nil {
				fmt.Fprint(w, "An error occured, please try again. Traceback:", err)
				return
			}
		}
		file, err := os.ReadFile("index.html")
		if err != nil {
			fmt.Fprint(w, "An error occured, please try again. Traceback:", err)
			return
		}
		html := strings.Replace(string(file), "Series found", "Chapters found", 1)
		html = strings.Replace(html, "--add here", chaptersB.String(), 1)
		fmt.Fprint(w, html)
	})

	// "/series/{series}/chapter-{chapter}" is invalid for some reason
	mux.HandleFunc("/series/{series}/{chapter}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		series := r.PathValue("series")
		var chapter uint8
		fmt.Sscanf(r.PathValue("chapter"), "chapter-%d", &chapter)
		file, err := os.ReadFile("videoplayer.html")
		if err != nil {
			fmt.Fprint(w, "An error occured, please try again. Traceback:", err)
			return
		}
		fmt.Print(r.Host)
		html := strings.Replace(string(file), "--replace here", fmt.Sprintf("http://%v/api/series/%v/chapter-%v", r.Host, series, chapter), 1)
		html = strings.Replace(html, "--replace es subtitles here", fmt.Sprintf("http://%v/api/series/%v/chapter-%v/captions?lang=spa", r.Host, series, chapter), 1)
		html = strings.Replace(html, "--replace en subtitles here", fmt.Sprintf("http://%v/api/series/%v/chapter-%v/captions", r.Host, series, chapter), 1)
		fmt.Fprint(w, html)
	})

	mux.HandleFunc("/api/series/{series}/{chapter}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		series := r.PathValue("series")
		var chapter int //cant use uint8 cuz i compare with len() later
		fmt.Sscanf(r.PathValue("chapter"), "chapter-%d", &chapter)
		entries, err := os.ReadDir("./series/" + series)
		if err != nil {
			fmt.Fprint(w, "An error occured, please try again. Traceback:", err)
			return
		}
		if chapter > len(entries) || chapter < 1 {
			fmt.Fprintf(w, "Chapter doesn't exist!")
			return
		}
		fmt.Println(entries[chapter-1].Name())
		http.ServeFile(w, r, fmt.Sprintf("./series/%v/%v", series, entries[chapter-1].Name()))
	})

	mux.HandleFunc("/api/series/{series}/{chapter}/captions", func(w http.ResponseWriter, r *http.Request) {
		series := r.PathValue("series")
		var chapter int
		fmt.Sscanf(r.PathValue("chapter"), "chapter-%d", &chapter)
		entries, err := os.ReadDir("./series/" + series)
		if err != nil {
			fmt.Fprint(w, "An error occured, please try again. Traceback:", err)
			return
		}
		if chapter > len(entries) || chapter < 1 {
			fmt.Fprintf(w, "Chapter doesn't exist!")
			return
		}
		fmt.Println(entries[chapter-1].Name())
		videoPath := fmt.Sprintf("./series/%v/%v", series, entries[chapter-1].Name())
		language := r.URL.Query().Get("lang")
		var langFormat string
		if language != "" {
			langFormat = "0:s:m:language:" + language
		} else {
			langFormat = "0:s:0" //1st lang
		}
		cmd := exec.Command("ffmpeg", "-i", videoPath, "-map", langFormat, "-f", "srt", "-")
		var stdout bytes.Buffer
		cmd.Stdout = &stdout
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		err = cmd.Run()
		if err != nil {
			// fmt.Print("error:", err, stderr.String()) // idc if it cant find captions
			w.WriteHeader(http.StatusNotFound)
			return
		}
		subtitles := stdout.String()
		vttHeader := "WEBVTT\n\n"
		re := regexp.MustCompile(`(\d{2}:\d{2}:\d{2}),(\d{3})`)
		vttSubtitles := re.ReplaceAllString(subtitles, "$1.$2")
		fmt.Fprint(w, vttHeader+vttSubtitles)
	})

	fmt.Println("Starting API server")
	host, err := os.Hostname()
	if err != nil {
		fmt.Println("An error occured. Traceback:", err)
		return
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		fmt.Println("An error occured. Traceback:", err)
		return
	}
	var hostIP string
	for _, ip := range ips {
		if ip.To4() != nil {
			hostIP = ip.String()
			break
		}
	}
	if hostIP == "" {
		fmt.Println("Couldn't find this machine's IPv4 address")
		return
	}
	fmt.Println("IPv4 is " + hostIP)
	if err := http.ListenAndServe("0.0.0.0:80", mux); err != nil {
		fmt.Println("Error occurred. Traceback:", err)
	}
}
