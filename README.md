次のようなコードで定期的に通知を送信

```go
package main

import (
	"log"
	"net/http"
	"time"
)

func sendHTTPRequest(url string, interval time.Duration) {
	ticker := time.NewTicker(interval)

	for {
		select {
		case <-ticker.C:
			resp, err := http.Get(url)
			if err != nil {
				log.Printf("Failed to send HTTP request: %s", err)
				continue
			}
			resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				log.Printf("Successfully sent HTTP request to %s", url)
			} else {
				log.Printf("Received HTTP status code %d from %s", resp.StatusCode, url)
			}
		}
	}
}

func main() {
	// この例では、5分（300秒）ごとに "http://localhost:8080" にGETリクエストを送ります。
	// URLとインターバルは適宜変更してください。
	go sendHTTPRequest("http://localhost:8080", 300*time.Second)

	// プログラムが終了しないようにするための処理
	select {}
}
```