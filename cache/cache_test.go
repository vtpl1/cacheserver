package cache

import (
	"io"
	"net/http"
	"sync"
	"testing"
	"time"
)

var urls = []string{
	"https://golang.org",
	"https://golang.org",
	"https://golang.org",
	"https://example.com",
	"https://www.wikipedia.org",
	"https://github.com/topics/python",
	"https://www.reddit.com/r/programming",
	"https://news.ycombinator.com",
	"https://www.amazon.com/dp/B08N5WRWNW",
	"https://stackoverflow.com/questions/12345678",
	"https://www.nytimes.com/2025/01/01/technology/tech-news.html",
	"https://www.imdb.com/title/tt1234567/",
	"https://www.youtube.com/watch?v=dQw4w9WgXcQ",
	"https://docs.python.org/3/library/random.html",
	"https://developer.mozilla.org/en-US/docs/Web/JavaScript",
	"https://www.linkedin.com/in/sample-profile",
	"https://twitter.com/elonmusk/status/1234567890123456789",
	"https://www.apple.com/ipad-pro/",
	"https://www.microsoft.com/en-us/software-download/windows10",
	"https://www.udemy.com/course/python-for-beginners/",
	"https://www.cnn.com/2025/01/01/world/global-news.html",
	"https://www.espn.com/nba/story/_/id/12345678/nba-finals-game-7",
	"https://www.ted.com/talks/jane_doe_how_to_build_resilience",
	"https://example.com",
	"https://www.wikipedia.org",
	"https://github.com/topics/python",
	"https://www.reddit.com/r/programming",
	"https://news.ycombinator.com",
	"https://www.amazon.com/dp/B08N5WRWNW",
	"https://stackoverflow.com/questions/12345678",
	"https://www.nytimes.com/2025/01/01/technology/tech-news.html",
	"https://www.imdb.com/title/tt1234567/",
	"https://www.youtube.com/watch?v=dQw4w9WgXcQ",
	"https://docs.python.org/3/library/random.html",
	"https://developer.mozilla.org/en-US/docs/Web/JavaScript",
	"https://www.linkedin.com/in/sample-profile",
	"https://twitter.com/elonmusk/status/1234567890123456789",
	"https://www.apple.com/ipad-pro/",
	"https://www.microsoft.com/en-us/software-download/windows10",
	"https://www.udemy.com/course/python-for-beginners/",
	"https://www.cnn.com/2025/01/01/world/global-news.html",
	"https://www.espn.com/nba/story/_/id/12345678/nba-finals-game-7",
	"https://www.ted.com/talks/jane_doe_how_to_build_resilience",
}

func httpGetBody(url string) func() (resultValue, error) {
	return func() (resultValue, error) {
		resp, err := http.Get(url)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		return io.ReadAll(resp.Body)
	}
}

func httpGetBody1(url string) (resultValue, error) {
	// fmt.Printf("Calling %s\n", url)
	if url == "https://www.wikipedia.org" {
		panic("Panic for " + url)
	}
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func incomingUrls(urls []string) <-chan string {
	out := make(chan string)
	go func() {
		for _, url := range urls {
			out <- url
		}
		close(out)
	}()
	return out
}

func TestConcurrent(t *testing.T) {
	startAll := time.Now()
	cache := NewCache(httpGetBody1)
	var n sync.WaitGroup
	for url := range incomingUrls(urls) {
		n.Add(1)
		go func(url string) {
			start := time.Now()
			value, err := cache.Get(url)
			if err != nil {
				t.Logf("%-25s %-15s error: %v\n", url, time.Since(start), err)
			} else {
				t.Logf("%-25s %-15s %d bytes\n", url, time.Since(start), len(value))
			}
			n.Done()
		}(url)
	}
	n.Wait()
	t.Logf("%-15s\n", time.Since(startAll))
}
