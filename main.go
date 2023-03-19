package main

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type urlset struct {
	XMLName xml.Name `xml:"urlset"`
	Urls    []url    `xml:"url"`
}

type url struct {
	XMLName xml.Name `xml:"url"`
	Loc     string   `xml:"loc"`
}
type urlData struct {
	URL  string `json:"url"`
	Post bool   `json:"post"`
}

var ctx = context.Background()

const jsonpath = "/Users/felix/Documents/openai/readmecreator/urls.json"

const sitemapUrl = "https://freegames.codes/sitemap.xml"

const readmePath = "/Users/felix/Documents/openai/readmecreator/readme.md"

func fetchH1Tags(url string) (string, string) {
	res, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	h1 := doc.Find("h1").First()

	return h1.Text(), url
}
func readJSONData(filename string) []map[string]interface{} {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}

	var jsonData []map[string]interface{}
	err = json.Unmarshal(data, &jsonData)
	if err != nil {
		panic(err)
	}

	return jsonData
}

func parseSitemap(sitemapUrl string) []map[string]interface{} {
	var urls []map[string]interface{}

	resp, err := http.Get(sitemapUrl)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return urls
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return urls
	}

	var s urlset
	err = xml.Unmarshal(data, &s)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return urls
	}

	for _, u := range s.Urls {
		urlsWithPost := map[string]interface{}{
			"url":  u.Loc,
			"post": false,
		}
		urls = append(urls, urlsWithPost)
	}

	return urls
}
func writeUrlsToJson(urls []map[string]interface{}, filename string) error {

	var existingUrls []map[string]interface{}
	if _, err := os.Stat(filename); !os.IsNotExist(err) {
		file, err := os.Open(filename)
		if err == nil {
			defer file.Close()
			decoder := json.NewDecoder(file)
			err = decoder.Decode(&existingUrls)
			if err != nil {
				return err
			}
		}
	}

	urlsMap := make(map[string]bool)
	for _, u := range existingUrls {
		urlsMap[u["url"].(string)] = true
	}
	for _, u := range urls {
		if !urlsMap[u["url"].(string)] {
			existingUrls = append(existingUrls, u)
		}
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "    ")
	err = encoder.Encode(existingUrls)
	if err != nil {
		return err
	}

	return nil
}

func appendToReadme(readmePath string, content string) error {
	f, err := os.OpenFile(readmePath, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err = f.WriteString("\n" + content); err != nil {
		return err
	}

	return nil
}
func main() {
	urls := parseSitemap(sitemapUrl)
	err := writeUrlsToJson(urls, jsonpath)
	if err != nil {
		panic(err)
	}

	urlData := readJSONData(jsonpath)

	updated := false
	for _, u := range urlData {
		if u["post"] == false {
			updated = true
			text, url := fetchH1Tags(u["url"].(string))
			fmt.Printf("%s: %s\n", url, text)

			markdownContent := fmt.Sprintf("[%s](%s)  \n", text, url)

			err := appendToReadme(readmePath, markdownContent)
			if err != nil {
				fmt.Println("Hata: ", err)
				return
			}

			fmt.Println("The Markdown file has been successfully added to the README file.")

			time.Sleep(1 * time.Second)
			u["post"] = true
		}
	}

	if updated == true {

		jsonData, err := json.Marshal(urlData)
		if err != nil {
			panic(err)
		}
		err = ioutil.WriteFile(jsonpath, jsonData, 0644)
		if err != nil {
			panic(err)
		}
	}

}
