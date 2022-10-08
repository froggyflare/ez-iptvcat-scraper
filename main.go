package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	curl "github.com/andelf/go-curl"
	"os"
	"regexp"
	"strconv"
	"strings"
	"bytes"

	app "iptvcat-scraper/pkg"

	"github.com/gocolly/colly"
)

const aHref = "a[href]"

func downloadFile(filepath string, url string) (err error) {
	fmt.Println("downloadFile from ", url, "to ", filepath)
	easy := curl.EasyInit()
	defer easy.Cleanup()

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()
		
	easy.Setopt(curl.OPT_URL, url)
	recv := func (buf []byte, userdata interface{}) bool {
		// Writer the body to file
		_, err = io.Copy(out, bytes.NewReader(buf))
		if err != nil {
			return false
		}
		return true
    }

	easy.Setopt(curl.OPT_WRITEFUNCTION, recv)

	// Get the data
    if err := easy.Perform(); err != nil {
        fmt.Printf("ERROR: %v\n", err)
    }

	if err != nil {
		return err
	}

	return nil
}

func getUrlFromFile(filepath string, origUrl string) (string, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	// Splits on newlines by default.
	scanner := bufio.NewScanner(f)

	line := 1
	// https://golang.org/pkg/bufio/#Scanner.Scan
	for scanner.Scan() {
		if strings.HasPrefix(strings.ToLower(scanner.Text()), "http") {
			return scanner.Text(), nil
		}
		line++
	}

	if err := scanner.Err(); err != nil {
		// Handle the error
	}

	return origUrl, err
}

func checkNestedUrls(skipOffline bool) {
	fmt.Println("checkNestedUrls()")

	converted_urls := map[string]string{}
	ignored := 0
	processed := 0

	for _, stream := range app.Streams.All {
		url_lower := strings.ToLower(stream.Link)

		// Check if need to skip offline, otherwise too many requests
		if skipOffline && (strings.ToLower(stream.Status) == "offline") {
			ignored++
			fmt.Println(">>> SKIP OFFLINE: ", ignored)
			continue
		}

		// Check for Minimum Liveliness, to avoid some extra scrapes
		minLiveliness := 80
		if streamLiveliness, err := strconv.Atoi(stream.Liveliness); err == nil {
			if streamLiveliness < minLiveliness {
				ignored++
				fmt.Println(">>> SKIP LIVELINESS, min, found: ", minLiveliness, streamLiveliness)
				continue
			}
		}

		if strings.Contains(url_lower, "list.iptvcat.com") {
			if _, ok := converted_urls[url_lower]; ok {
				// stream.Link = converted_urls[url_lower]
				ignored++
				fmt.Println(">>> SKIP DUPLICATE: ", ignored)
				continue
			}

			const tmpFile = "tmp.m3u8"
			// Download the file
			downloadFile(tmpFile, stream.Link)

			// Get the Url
			newUrl, err := getUrlFromFile(tmpFile, stream.Link)
			if err != nil {
				fmt.Println(err)
				//return
			}
			//fmt.Println("newUrl found in link: ", newUrl)
			stream.Link = newUrl
			converted_urls[url_lower] = newUrl

			// Add M3du
			m3du, err := os.ReadFile(tmpFile)
			if err != nil {
				fmt.Println(err)
			}
			stream.M3DU = string(m3du)

			processed++

			// Delete the file
			err2 := os.Remove(tmpFile)
			if err2 != nil {
				fmt.Println(err2)
				return
			}

		} else {
			fmt.Println("no m3u8 found in link: ", stream.Link)
		}
	}

	fmt.Println("### MAP ", converted_urls)
	fmt.Println("### ignored ", ignored)
	fmt.Println("### processed ", processed)

}

func writeToFile() {
	streamsAll, err := json.MarshalIndent(app.Streams.All, "", "    ")
	streamsCountry, err := json.MarshalIndent(app.Streams.ByCountry, "", "    ")
	if err != nil {
		fmt.Println("error:", err)
	}

	os.MkdirAll("data/countries", os.ModePerm)

	ioutil.WriteFile("data/all-streams.json", streamsAll, 0644)
	ioutil.WriteFile("data/all-by-country.json", streamsCountry, 0644)
	for key, val := range app.Streams.ByCountry {
		// streamsCountry, err := json.Marshal(val)
		streamsCountry, err := json.MarshalIndent(val, "", "    ")
		if err != nil {
			fmt.Println("error:", err)
		}
		ioutil.WriteFile("data/countries/"+key+".json", streamsCountry, 0644)
	}

	f, err := os.Create("data/all-streams.m3du")
	if err != nil {
		fmt.Println("error opening m3du file:", err)
	}

	for _, stream := range app.Streams.All {
		f.Write([]byte(stream.M3DU))
	}
	
	f.Close()
}

func processUrl(url string, domain string, skipOffline bool) {
	urlFilters := regexp.MustCompile(url + ".*")
	c := colly.NewCollector(
		colly.AllowedDomains(domain),
		colly.URLFilters(urlFilters),
		colly.Async(true),
	)

	c.OnResponse(func(r *colly.Response) {
		fmt.Println("Visited", r.Request.URL)
	})

	c.OnHTML(aHref, app.HandleFollowLinks(c))
	c.OnHTML(app.GetStreamTableSelector(), app.HandleStreamTable(c))

	c.OnScraped(func(r *colly.Response) {
		fmt.Println("Finished", r.Request.URL)
	})

	c.OnError(func(r *colly.Response, err error) {
		fmt.Printf("Error: %d %s\n", r.StatusCode, r.Request.URL)
	})

	c.Visit(url)
	c.Wait()
	checkNestedUrls(skipOffline)
	writeToFile()
}

func main() {
	const iptvCatDomain = "iptvcat.com"

	urlList := [...]string{
		"https://iptvcat.com/australia",
		//"https://iptvcat.com/austria",
		"https://iptvcat.com/canada",
		//"https://iptvcat.com/germany",
		//"https://iptvcat.com/switzerland",
		"https://iptvcat.com/united_states_of_america",
		// "https://iptvcat.com/china",
		// "https://iptvcat.com/hong_kong",
		// "https://iptvcat.com/india",
		"https://iptvcat.com/iraq",
		"https://iptvcat.com/egypt",
		"https://iptvcat.com/saudi_arabia",
		"https://iptvcat.com/kuwait",
		"https://iptvcat.com/israel",
		"https://iptvcat.com/jordan",
		"https://iptvcat.com/syria",
		"https://iptvcat.com/lebanon",
		"https://iptvcat.com/qatar",
		"https://iptvcat.com/united_arab_emirates",
		"https://iptvcat.com/bahrain",
		"https://iptvcat.com/united_kingdom",
		// "https://iptvcat.com/japan",
		// "https://iptvcat.com/malaysia",
		// "https://iptvcat.com/singapore",
		// "https://iptvcat.com/south_korea",
		// "https://iptvcat.com/belgium",
		// "https://iptvcat.com/france",
		// "https://iptvcat.com/ireland",
		// "https://iptvcat.com/italy",
		// "https://iptvcat.com/new_zealand",
	}

	for _, element := range urlList {
		processUrl(element, iptvCatDomain, true)
	}

}
