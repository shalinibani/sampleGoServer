package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"time"

	"github.com/patrickmn/go-cache"
)

type payload struct {
	Numbers []int `json:"numbers"`
}

type result struct {
	data payload
}

const (
	timeoutServer   = 500 * time.Millisecond // total timeout of the server
	timeoutGetReq   = 400 * time.Millisecond // total timeout when sending GET request for each given URL
	cacheExpiration = 10 * time.Minute
)

const errServer = "error occurred in server"

type client struct {
	cache cache.Cache
	urls  []string
}

var serverCache = cache.New(cacheExpiration, 0)

func main() {

	http.HandleFunc("/number", numbersHandler)

	server := http.Server{
		Addr:    ":8080",
		Handler: http.TimeoutHandler(http.HandlerFunc(numbersHandler), timeoutServer, "server timeout!"),
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("%s: %s", errServer, err)
	}
}

func numbersHandler(w http.ResponseWriter, r *http.Request) {
	log.Print("processing the request")

	if r.URL.Path != "/numbers" {
		http.Error(w, "404 not found", http.StatusNotFound)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "method is not supported", http.StatusNotFound)
		return
	}

	ctx := context.Background()

	queryValues := r.URL.Query()

	urls := queryValues["u"]

	results := processURLs(ctx, urls)
	sortedNumbers := processFinalResult(results)

	respPayload := payload{Numbers: sortedNumbers}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(respPayload); err != nil {
		log.Fatalf("%s: %s", errServer, err)
	}
}

// processURLs goes through the list of input URLs and queries each one of them
func processURLs(ctx context.Context, urls []string) []result {
	var results []result

	ch := make(chan result)

	childCtx, cancel := context.WithTimeout(ctx, timeoutGetReq)
	defer cancel()

	for _, url := range urls {
		go getResponseFromURL(childCtx, ch, url)
	}

	for i := 0; i < len(urls); i++ {
		results = append(results, <-ch)
	}

	return results
}

func getResponseFromURL(ctx context.Context, ch chan result, url string) {
	select {
	case <-ctx.Done():
		log.Printf("timeout reached for URL: %s. Fetching result from cache...", url)
		ch <- result{data: payload{Numbers: getResultFromCache(url)}}
		return
	default:
		makeGetRequestForURL(ctx, url, ch)
	}
}

func getResultFromCache(url string) []int {
	var data []int

	val, ok := serverCache.Get(url)
	if ok {
		data, _ = val.([]int) // not required to check the bool value as we insert only []int type in cache.
	}

	return data
}

func updateCache(url string, numbers []int) {
	serverCache.Delete(url)
	serverCache.Add(url, numbers, cacheExpiration)
}

// makeGetRequestForURL makes a GET request to the given URL and returns the result in the given channel.
func makeGetRequestForURL(ctx context.Context, url string, ch chan result) {
	log.Printf("sending GET request to URL %s", url)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		// If an error is encountered, ignore the error and return cached result
		log.Printf("%s. Fetching result from cache", err)
		ch <- result{data: payload{Numbers: getResultFromCache(url)}}

		return
	}

	client := http.DefaultClient

	var numbers payload

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("%s. Fetching result from cache", err)
		ch <- result{data: payload{Numbers: getResultFromCache(url)}}

		return
	}

	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&numbers); err != nil {
		// If an error is encountered, ignore the error and return cached result
		log.Printf("%s. Fetching result from cache", err)
		ch <- result{data: payload{Numbers: getResultFromCache(url)}}

		return
	}

	// if we successfully reach till here, and the returned numbers from URL is not empty, then update cache.
	if len(numbers.Numbers) > 0 {
		updateCache(url, numbers.Numbers)
	}

	ch <- result{data: numbers}
}

// processFinalResult processes the result returned by each URL. It removes the duplicate, sorts them and returns a
// slice of sorted int.
func processFinalResult(res []result) []int {
	uniqueNumbers := make(map[int]struct{})
	uniqueNumbersSlice := make([]int, 0, len(res))

	for _, r := range res {
		for _, num := range r.data.Numbers {
			// check if the number is unique. If it is unique then only append it to the final int slice.
			_, ok := uniqueNumbers[num]
			if !ok {
				uniqueNumbers[num] = struct{}{}
				uniqueNumbersSlice = append(uniqueNumbersSlice, num)
			}
		}
	}

	sort.Ints(uniqueNumbersSlice)

	return uniqueNumbersSlice
}
