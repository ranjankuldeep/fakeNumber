package runner

import (
	"log"
	"net/http"
	"time"
)

func makeRequestWithRetries(url string, retries int) bool {
	for i := 0; i < retries; i++ {
		resp, err := http.Get(url)
		if err == nil && resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()

			buf := make([]byte, 2)
			n, _ := resp.Body.Read(buf)
			if string(buf[:n]) == "ok" {
				return true
			}

			log.Printf("Unexpected response for %s: %s", url, string(buf[:n]))
		} else {
			log.Printf("Error calling %s: %v", url, err)
		}

		time.Sleep(2 * time.Second)
	}

	return false
}

func StartUrlCallTicker(urls []string) {
	go func() {
		for {
			now := time.Now()
			nextMidnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
			durationUntilMidnight := time.Until(nextMidnight)
			time.Sleep(durationUntilMidnight)

			log.Println("Starting midnight task...")
			for _, url := range urls {
				success := makeRequestWithRetries(url, 3)
				if success {
					log.Printf("Successfully called %s", url)
				} else {
					log.Printf("Failed to get a valid response from %s after retries", url)
				}
			}
		}
	}()
}
