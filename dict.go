package main

import (
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"code.google.com/p/go.text/unicode/norm"
	"github.com/PuerkitoBio/goquery"
)

const (
	bundleId       = "alfred2-dictcc-lookup"
	volatileDir    = "Library/Caches/com.runningwithcrayons.Alfred-2/Workflow Data"
	nonVolatileDir = "Library/Application Support/Alfred 2/Workflow Data"
	suggestUrl     = "http://www.dict.cc/inc/ajax_autosuggest.php"
)

var (
	iconpaths = map[int]string{
		1: "Icons/de.png",
		2: "Icons/en.png"}
	urlformat = map[int]string{
		1: "http://www.dict.cc/deutsch-englisch/%s.html",
		2: "http://www.dict.cc/englisch-deutsch/%s.html"}
	previewTimeout int
	maxConcurrent  int
)

type Item struct {
	XMLName      xml.Name `xml:"item"`
	Uid          string   `xml:"uid,attr,omitempty"`
	Arg          string   `xml:"arg,attr,omitempty"`
	Autocomplete string   `xml:"autocomplete,attr,omitempty"`
	Title        string   `xml:"title"`
	Subtitle     string   `xml:"subtitle,omitempty"`
	Icon         string   `xml:"icon,omitempty"`
}

func parallelize(n int, maxConcurrent int, f func(int)) {
	mc := maxConcurrent
	if mc <= 0 {
		mc = n
	}
	started := make(chan bool, mc)
	finished := make(chan bool, n)
	for i := 0; i < n; i++ {
		go func(i int) {
			started <- true
			f(i)
			<-started
			finished <- true
		}(i)
	}
	for i := 0; i < n; i++ {
		<-finished
	}
}

func withTimeout(t time.Duration, f func()) {
	done := make(chan bool, 1)
	go func() {
		f()
		done <- true
	}()
	select {
	case <-done:
	case <-time.After(t):
	}
	return
}

func makeItem(word, preview string, lang int) Item {
	return Item{
		Title:    word,
		Subtitle: preview,
		Icon:     iconpaths[lang],
		Arg:      fmt.Sprintf(urlformat[lang], url.QueryEscape(word))}
}

func suggestions(s string) (suggs []string, err error) {
	resp, err := http.PostForm(suggestUrl, url.Values{"s": {s}})
	if err != nil {
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	bodystring := strings.TrimRight(string(body), "\n ")
	if bodystring != "" {
		suggs = strings.Split(bodystring, "\n")
	}
	return
}

func getTranslations(ds *Datastore, word string, lang int) (transstr string) {
	key := string(lang) + "|" + strings.ToLower(word)
	transstr, err := ds.Get(key)
	if err != nil {
		trans, err := scrapeTranslations(word, lang)
		if err != nil {
			transstr = ""
		} else {
			transstr = strings.Join(trans, " · ")
			ds.Set(key, transstr)
		}
	}
	return
}

func scrapeTranslations(word string, lang int) (trans []string, err error) {
	err = errors.New("timeout")
	withTimeout(time.Duration(previewTimeout)*time.Millisecond, func() {
		doc, err2 := goquery.NewDocument(
			fmt.Sprintf(urlformat[lang], word))
		if err2 != nil {
			return
		}
		trans = doc.Find("dd").First().Find("a").Map(
			func(i int, s *goquery.Selection) string {
				return s.Text()
			})
		err = nil
	})
	return
}

func parseArgs() (word string, useVolatileStore bool) {
	flag.IntVar(
		&previewTimeout,
		"timeout",
		500,
		"timeout for preview requests.")
	flag.StringVar(&word, "word", "", "word to translate")
	flag.IntVar(
		&maxConcurrent,
		"maxconcurrent",
		4,
		"maximum number of concurrent preview requests "+
			"(negative value for unlimited)")
	flag.BoolVar(
		&useVolatileStore,
		"volatilestore",
		false,
		"use a volatile store location for caching previews")
	flag.Parse()
	word = norm.NFC.String(word)
	return
}

func main() {
	word, useVolatileStore := parseArgs()

	usr, _ := user.Current()
	var dir string
	if useVolatileStore {
		dir = filepath.Join(usr.HomeDir, volatileDir, bundleId)
	} else {
		dir = filepath.Join(usr.HomeDir, nonVolatileDir, bundleId)
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		panic(err)
	}
	ds, err := OpenDatastore(filepath.Join(dir, "cache.gob"))
	if err != nil {
		panic(err)
	}
	defer ds.Close()

	lines, err := suggestions(word)
	if err != nil {
		panic(err)
	}
	items := make([]Item, len(lines))
	parallelize(len(lines), maxConcurrent, func(i int) {
		line := strings.Split(lines[i], "\t")
		word := line[0]
		lang, err := strconv.Atoi(line[1])
		if err != nil {
			panic("Malformed response")
		}
		preview := getTranslations(ds, word, lang)
		items[i] = makeItem(word, preview, lang)
	})

	enc := xml.NewEncoder(os.Stdout)
	enc.Indent("    ", "    ")
	fmt.Println("<?xml version=\"1.0\"?>\n<items>")
	if err := enc.Encode(items); err != nil {
		panic(err)
	}
	fmt.Println("\n</items>")
}
