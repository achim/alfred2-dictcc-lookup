package main

import (
	"encoding/xml"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/fsouza/gokabinet/kc"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	dataDir = "Library/Caches/com.runningwithcrayons.Alfred-2/" +
		"Workflow Data/alfred2-dictcc-lookup"
	autosuggestUrl = "http://www.dict.cc/inc/ajax_autosuggest.php"
)

var (
	iconpaths = map[int]string{
		1: "Icons/de.png",
		2: "Icons/en.png"}
	label = map[int]string{
		1: "de → en",
		2: "en → de"}
	urlformat = map[int]string{
		1: "http://www.dict.cc/deutsch-englisch/%s.html",
		2: "http://www.dict.cc/englisch-deutsch/%s.html"}
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

func makeItem(db *kc.DB, word string, lang int) Item {
	return Item{
		Title:    word + " [" + label[lang] + "]",
		Subtitle: getTranslations(db, word, lang),
		Icon:     iconpaths[lang],
		Arg:      fmt.Sprintf(urlformat[lang], word)}
}

func suggestions(db *kc.DB, s string) (suggs []Item, err error) {
	resp, err := http.PostForm(autosuggestUrl, url.Values{"s": {s}})
	if err != nil {
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	bodystring := strings.TrimRight(string(body), "\n ")
	lines := strings.Split(bodystring, "\n")
	resultlen := 0
	if bodystring != "" {
		resultlen = len(lines)
	}
	suggs = make([]Item, resultlen)
	var (
		line []string
		lang int
	)
	c := make(chan bool, resultlen)
	for i := range lines {
		go func(i int) {
			line = strings.Split(lines[i], "\t")
			lang, err = strconv.Atoi(line[1])
			if err != nil {
				panic("Malformed response.")
			}
			suggs[i] = makeItem(db, line[0], lang)
			c <- true
		}(i)
	}
	for _ = range suggs {
		<-c
	}
	return
}

func getTranslations(db *kc.DB, word string, lang int) (transstr string) {
	key := string(lang) + "|" + strings.ToLower(word)
	transstr, err := db.Get(key)
	if err != nil {
		trans := scrapeTranslations(word, lang)
		transstr = strings.Join(trans, " · ")
		if len(trans) > 0 {
			db.Set(key, transstr)
		}
	}
	return
}

func scrapeTranslations(word string, lang int) (trans []string) {
	done := make(chan bool, 1)
	timeout := time.After(500 * time.Millisecond)
	go func() {
		doc, err := goquery.NewDocument(fmt.Sprintf(urlformat[lang], word))
		if err != nil {
			panic(err)
		}
		trans = doc.Find("dd").First().Find("a").Map(
			func(i int, s *goquery.Selection) string {
				return s.Text()
			})
		done <- true
	}()
	select {
	case <- done:
	case <- timeout:
	}
	return
}

func main() {
	if len(os.Args) != 2 {
		panic("usage: dict <word>")
	}

	usr, _ := user.Current()
	dir := filepath.Join(usr.HomeDir, dataDir)

	if err := os.MkdirAll(dir, 0755); err != nil {
		panic(err)
	}
	db, err := kc.Open(filepath.Join(dir, "cache.kch"), kc.WRITE)
	if err != nil {
		panic(err)
	}
	defer db.Close()
	suggs, err := suggestions(db, os.Args[1])
	if err != nil {
		panic(err)
	}

	enc := xml.NewEncoder(os.Stdout)
	enc.Indent("    ", "    ")
	fmt.Println("<?xml version=\"1.0\"?>\n<items>")
	if err := enc.Encode(suggs); err != nil {
		panic(err)
	}
	fmt.Println("\n</items>")
}
