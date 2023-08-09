package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

type Website struct {
	URL              string
	ResponseCode     int
	ResponseBodySize int
	ScreenshotData   []byte
}

func main() {
	urlFile := flag.String("urlfile", "", "File containing URLs")
	logFile := flag.String("logfile", "", "File to log the results")
	flag.Parse()

	urls, err := readLines(*urlFile)
	if err != nil {
		fmt.Printf("Failed to read file: %s", err)
		os.Exit(1)
	}

	var websites []Website
	for _, url := range urls {
		fmt.Printf("Processing URL: %s\n", url)
		website := fetchWebsite(url)
		if website != nil {
			websites = append(websites, *website)
		}
	}

	writeLog(*logFile, websites)
}

func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func fetchWebsite(url string) *Website {
	ctx, cancel := chromedp.NewContext(context.Background(), chromedp.WithLogf(log.Printf))
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	var responseReceived network.EventResponseReceived
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		switch ev := ev.(type) {
		case *network.EventResponseReceived:
			responseReceived = *ev
		}
	})

	var screenshot []byte
	var body string
	err := chromedp.Run(ctx,
		network.Enable(),
		chromedp.Navigate(url),
		chromedp.CaptureScreenshot(&screenshot),
		chromedp.OuterHTML("html", &body),
	)
	if err != nil {
		fmt.Printf("Failed to capture screenshot for %s: %v", url, err)
		return nil
	}

	fileName := getFileName(url)
	err = ioutil.WriteFile(fileName, screenshot, 0644)
	if err != nil {
		fmt.Printf("Failed to write screenshot to file for %s: %v", url, err)
		return nil
	}

	website := Website{
		URL:               url,
		ResponseCode:      int(responseReceived.Response.Status),
		ResponseBodySize:  len(body),
		ScreenshotData:    screenshot,
	}
	return &website
}

func getFileName(url string) string {
	fileName := strings.ReplaceAll(url, "http://", "")
	fileName = strings.ReplaceAll(fileName, "https://", "")
	fileName = strings.ReplaceAll(fileName, "/", "_")
	return fileName + ".png"
}

func writeLog(logFile string, websites []Website) {
	f, err := os.Create(logFile)
	if err != nil {
		fmt.Printf("Failed to create log file: %s", err)
		os.Exit(1)
	}
	defer f.Close()

	f.WriteString("<table><tr><th>URL</th><th>Response Code</th><th>Body Size</th><th>Screenshot</th></tr>")
	for _, website := range websites {
		f.WriteString(fmt.Sprintf("<tr><td><a href='%s'>%s</a></td><td>%d</td><td>%d</td><td><a href='%s'>Link</a></td></tr>", 
		website.URL, website.URL, website.ResponseCode, website.ResponseBodySize, getFileName(website.URL)))
	}
	f.WriteString("</table>")
}
