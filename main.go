package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"flag"

	"github.com/PuerkitoBio/goquery"
	"github.com/fatih/color"
	"golang.org/x/time/rate"
)

func main() {
	additional := flag.String("additional", "", "additional-json")
	flag.Parse()

	if len(flag.Args()) == 1 {
		fmt.Printf("%s [event number]\n", os.Args[0])
	}

	eventNumber := flag.Arg(0)

	os.MkdirAll("output/cache", 0755)
	os.MkdirAll("output/pages", 0755)
	os.MkdirAll("output/images", 0755)

	var cache *Cache

	jsonFilePath := fmt.Sprintf("output/cache/%s.json", eventNumber)
	cacheFile, err := os.Open(jsonFilePath)

	if err == nil {
		defer cacheFile.Close()
		dec := json.NewDecoder(cacheFile)
		cache = &Cache{}
		err = dec.Decode(cache)
	} else {
		cache, err = readConnpass(eventNumber)
	}
	if err != nil {
		panic(err)
	}
	if *additional != "" {
		addFile, err := os.Open(*additional)
		if err != nil {
			panic(err)
		}
		defer addFile.Close()
		extraFiles := &Cache{}
		dec := json.NewDecoder(addFile)
		err = dec.Decode(extraFiles)
		if err != nil {
			panic(err)
		}
		cache.Pages = append(cache.Pages, extraFiles.Pages...)
	}
	for _, page := range cache.Pages {
		fileName := strings.ReplaceAll(page.Category, " ", "")
		fileName = strings.ReplaceAll(page.Category, "/", "_")
		page.Draw(cache.EventName, fmt.Sprintf("output/pages/%s.pdf", fileName))
	}
}

func readConnpass(eventNumber string) (*Cache, error) {
	res, err := http.Get(fmt.Sprintf("https://connpass.com/event/%s/participation/", eventNumber))
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, err
	}
	spaces := regexp.MustCompile("\\s+")

	limiter := rate.NewLimiter(rate.Every(time.Second), 1)

	var cache Cache
	cache.EventName = doc.Find(".event_title").Text()

	doc.Find("table.participants_table").Each(parseTable(spaces, limiter, &cache))
	jsonFilePath := fmt.Sprintf("output/cache/%s.json", eventNumber)
	cacheFile, err := os.Create(jsonFilePath)
	if err != nil {
		return nil, err
	}
	defer cacheFile.Close()
	encoder := json.NewEncoder(cacheFile)
	encoder.Encode(&cache)
	return &cache, nil
}

func parseTable(spaces *regexp.Regexp, limiter *rate.Limiter, cache *Cache) func(index int, s *goquery.Selection) {
	return func(index int, s *goquery.Selection) {
		title := spaces.ReplaceAllString(s.Find("thead tr th").Text(), " ")
		color.Green(title)
		page := Page{
			Category: title,
		}
		s.Find(".display_name > a").Each(parseMember(limiter, &page))
		more := s.Find("tbody tr.empty")
		if len(more.Nodes) > 0 {
			link, ok := more.Find("a").Attr("href")
			if ok {
				res, err := http.Get(link)
				if err != nil {
					log.Fatalln(err)
				}
				doc, err := goquery.NewDocumentFromReader(res.Body)
				if err != nil {
					log.Fatalln(err)
				}
				var queries []string
				doc.Find(".paging_area a").Each(func(index int, s *goquery.Selection) {
					if href, ok := s.Attr("href"); ok {
						if index == 0 || queries[len(queries)-1] != href {
							queries = append(queries, href)
						}
					}
				})
				for _, query := range queries {
					res, err := http.Get(link + query)
					if err != nil {
						log.Fatalln(err)
					}
					doc, err := goquery.NewDocumentFromReader(res.Body)
					if err != nil {
						log.Fatalln(err)
					}
					doc.Find(".display_name > a").Each(parseMember(limiter, &page))
				}
			}
		}
		cache.Pages = append(cache.Pages, page)
	}
}

func parseMember(limiter *rate.Limiter, page *Page) func(index int, s *goquery.Selection) {
	return func(index int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		fragments := strings.Split(href, "/")
		name := fragments[len(fragments)-2]
		if name == "open" {
			name = fragments[len(fragments)-3]
		}
		color.Cyan(s.Text() + " (" + href + ")")
		imagepath := downloadImage(name, href, limiter)
		user := Card{
			Name:      s.Text(),
			ImagePath: imagepath,
		}
		page.Cards = append(page.Cards, user)
	}
}

func downloadImage(name string, href string, limiter *rate.Limiter) string {
	imagepath := filepath.Join("output/images", name+".png")
	_, err := os.Lstat(imagepath)
	if os.IsNotExist(err) {
		for i := 0; i < 10; i++ {
			res, err := http.Get(href)
			if err != nil {
				log.Printf("@1 error %s", err.Error())
				continue
			}
			defer res.Body.Close()
			profile, err := goquery.NewDocumentFromReader(res.Body)
			if err != nil {
				log.Printf("@2 error %s", err.Error())
				continue
			}
			img := profile.Find("#side_area > div.mb_20.text_center > a > img")
			source, _ := img.Attr("src")
			if source == "" {
				source = "https://connpass.com/static/img/common/user_no_image_180.png"
			}
			ires, err := http.Get(source)
			if err != nil {
				log.Printf("@3 error %s", err.Error())
				continue
			}
			i, err := os.Create(imagepath)
			defer ires.Body.Close()
			io.Copy(i, ires.Body)
			break
		}
		limiter.Wait(context.Background())
	}
	return imagepath
}
