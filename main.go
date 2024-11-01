package main

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/RomainMichau/cloudscraper_go/cloudscraper"
	"github.com/sfomuseum/go-exif-update"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

var basePath = "C:\\jav"
var pageIndex = 1
var totalPhoto = make(map[string]string)
var deletePhoto []string

func getActressList(url string, localPage int) {
	client, _ := cloudscraper.Init(false, false)
	requestUrl := fmt.Sprintf("%s&page=%d", url, localPage)
	res, _ := client.Get(requestUrl, make(map[string]string), "")

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(res.Body))
	if err != nil {
		fmt.Println("get请求失败：", err)
	}

	doc.Find("#price-currency").Each(func(i int, s *goquery.Selection) {
		str := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(s.Text(), "\n", ""), " ", ""), "/", "")
		if len(str) > 0 {
			if atoi, err := strconv.Atoi(str); err == nil {
				pageIndex = atoi
			}
		}
	})

	doc.Find("a[class^=text-secondary]").Each(func(i int, s *goquery.Selection) {
		// For each item found, get the title
		link, _ := s.Attr("href")
		number, _ := s.Attr("alt")
		if len(link) > 0 && len(number) > 0 {
			totalPhoto[number] = link
		}
	})

	if pageIndex > localPage {
		getActressList(url, localPage+1)
	}
}

func getMovieInfo(url string, number string) {
	client, _ := cloudscraper.Init(false, false)
	res, _ := client.Get(url, make(map[string]string), "")

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(res.Body))
	if err != nil {
		fmt.Println("get请求失败：", err)
	}

	maxSize := 0.00
	maxLink := ""

	doc.Find("tbody[class^=divide-y]>tr").Each(func(i int, s *goquery.Selection) {
		// For each item found, get the title
		s.Find(".whitespace-nowrap.pl-4.text-right.text-sm.text-gray-400.font-mono").Each(func(j int, m *goquery.Selection) {
			if strings.HasSuffix(m.Text(), "GB") {
				if atoi, err := strconv.ParseFloat(strings.ReplaceAll(m.Text(), "GB", ""), 64); err == nil {
					if atoi > maxSize {
						s.Find("a[rel=nofollow]").Each(func(z int, d *goquery.Selection) {
							url, _ := d.Attr("href")
							if len(url) > 0 {
								maxLink = url
							}
						})
						maxSize = atoi
					}
				}
			}
		})
	})

	if maxLink != "" {
		photoUrl := fmt.Sprintf("https://fivetiu.com/%s/cover-n.jpg", number)
		res, err := http.Get(photoUrl)
		if err != nil {
			return
		}
		if res == nil {
			return
		}
		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			return
		}

		movieTime := "2000-01-01"
		doc.Find("div>time").Each(func(i int, s *goquery.Selection) {
			if len(s.Text()) > 0 {
				movieTime = s.Text()
			}
		})

		filePath := fmt.Sprintf("%s/%s-%s.jpg", basePath, movieTime, number)

		f, err := os.Create(filePath)
		if err != nil {
			return
		}
		_, err = io.Copy(f, res.Body)
		if err != nil {
			return
		}

		exifProps := map[string]interface{}{
			"Artist": maxLink,
		}

		source, _ := os.Open(filePath)
		defer source.Close()
		bakFilePath := fmt.Sprintf("%s.bak", filePath)
		out, _ := os.Create(bakFilePath)
		defer out.Close()

		_ = update.PrepareAndUpdateExif(source, out, exifProps)

		deletePhoto = append(deletePhoto, fmt.Sprintf("%s-%s", movieTime, number))
	}
	time.Sleep(1 * time.Second)
}

func main() {
	getActressList("https://missav.com/dm51/actresses/%E6%9D%BE%E4%B8%8B%E7%B4%97%E6%A6%AE%E5%AD%90?filters=individual", 1)

	fmt.Printf("total video is %d\n", len(totalPhoto))

	for key, value := range totalPhoto {
		getMovieInfo(value, key)
	}

	time.Sleep(10 * time.Second)

	for _, value := range deletePhoto {
		filePath := fmt.Sprintf("%s/%s.jpg", basePath, value)
		bakFilePath := fmt.Sprintf("%s.bak", filePath)
		_ = os.Remove(filePath)
		_ = os.Rename(bakFilePath, filePath)
	}
}
