package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"

	httpstat "github.com/tcnksm/go-httpstat"
)

var wg sync.WaitGroup

func worker(id, wl, interval int, url, body, method string, results chan<- httpstat.Result) {
	defer wg.Done()
	for w := 1; w <= wl; w++ {
		fmt.Printf("\033[2K\r Worker %d \x1b[33mstarted\x1b[0m job %d", id, w)
		result := doRequest(url, body, method)
		fmt.Printf("\033[2K\r Worker %d \x1b[32mfinished\x1b[0m job %d", id, w)
		results <- result
		time.Sleep(time.Duration(interval * int(time.Second)))
	}
}

func doRequest(url, body, method string) httpstat.Result {
	// Create a new HTTP request
	req, err := http.NewRequest(method, url, bytes.NewBuffer([]byte(body)))

	// Create a httpstat powered context
	var result httpstat.Result
	ctx := httpstat.WithHTTPStat(req.Context(), &result)
	req = req.WithContext(ctx)

	// Send request by default HTTP client
	client := http.DefaultClient
	res, err := client.Do(req)
	result.End(time.Now())

	if err != nil {
		log.Fatal(err)
	}
	if _, err := io.Copy(ioutil.Discard, res.Body); err != nil {
		log.Fatal(err)
	}
	res.Body.Close()
	// end := time.Now()

	return result
}

func main() {
	url := flag.String("url", "", "url to deliver payload to")
	body := flag.String("body", "", "Body to be delivered")
	get := flag.Bool("get", false, "use get request. Default = False")
	wo := flag.Int("wo", 10, "Worker Count: amount of workers making simultaneos requests")
	wl := flag.Int("wl", 10, "Worker Load: amount of requests executed per worker")
	interval := flag.Int("interval", 0, "Interval in seconds between each iteration for each worker")

	flag.Parse()

	var method string
	if *get {
		method = "GET"

	} else {
		method = "POST"
	}

	results := make(chan httpstat.Result, *wo**wl)
	ts := time.Now()
	for w := 1; w <= *wo; w++ {
		wg.Add(1)
		go worker(w, *wl, *interval, *url, *body, method, results)
	}

	var avgDNS, avgTCP, avgTLS, avgServ, avgTransfer, avgConnect float64
	var amountRecieved float64

	wg.Wait()
	tt := time.Since(ts) //ts - time.Now().
	for i := 0; i < *wo**wl; i++ {
		select {
		case result := <-results:

			avgDNS += float64(result.DNSLookup / time.Millisecond)
			avgConnect += float64(result.Connect / time.Millisecond)
			avgTCP += float64(result.TCPConnection / time.Millisecond)
			avgTLS += float64(result.TLSHandshake / time.Millisecond)
			avgServ += float64(result.ServerProcessing / time.Millisecond)
			avgTransfer += float64(result.ServerProcessing / time.Millisecond)
			amountRecieved++

		default:
			log.Println("Test payload failed")
		}
	}
	fmt.Printf("\n")
	log.Printf("Average DNS lookup: %f ms", avgDNS/amountRecieved)
	log.Printf("Average Connection: %f ms", avgConnect/amountRecieved)
	log.Printf("Average TLS handshake: %f ms", avgTLS/amountRecieved)
	log.Printf("Average Server processing: %f ms", avgServ/amountRecieved)
	log.Printf("Average Content transfer: %f ms", avgTransfer/amountRecieved)
	log.Printf("Total test duration: %f ms", float64(tt/time.Microsecond))
}
