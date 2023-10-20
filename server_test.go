package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestProcessURLsTimeout(t *testing.T) {
	t.Run("no timeout", func(t *testing.T) {
		rq := require.New(t)

		respPayload1 := payload{Numbers: []int{5, 7}}
		respPayload2 := payload{Numbers: []int{1, 2}}

		server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			err := json.NewEncoder(w).Encode(respPayload1)

			rq.NoError(err)
		}))

		defer server1.Close()

		server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			err := json.NewEncoder(w).Encode(respPayload2)

			rq.NoError(err)
		}))

		defer server2.Close()

		results := processURLs(context.Background(), []string{server1.URL, server2.URL})
		rq.Equal(2, len(results))

		expectedRes := map[int]struct{}{
			1: {},
			2: {},
			5: {},
			7: {},
		}

		for _, res := range results {
			for _, v := range res.data.Numbers {
				_, ok := expectedRes[v]
				rq.True(ok)
			}
		}
	})

	t.Run("timeout - cached result", func(t *testing.T) {
		rq := require.New(t)

		respPayload1 := payload{Numbers: []int{5, 7}}
		respPayload2 := payload{Numbers: []int{1, 2}}

		server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			err := json.NewEncoder(w).Encode(respPayload1)

			rq.NoError(err)
		}))

		defer server1.Close()

		server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			err := json.NewEncoder(w).Encode(respPayload2)

			rq.NoError(err)
		}))

		defer server2.Close()

		//add some values to cache
		updateCache(server1.URL, []int{1, 5})
		updateCache(server2.URL, []int{6, 9})

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		cancel() // send a cancelled context

		results := processURLs(ctx, []string{server1.URL, server2.URL})
		rq.Equal(2, len(results))

		expectedRes := map[int]struct{}{
			1: {},
			5: {},
			6: {},
			9: {},
		}

		for _, res := range results {
			for _, v := range res.data.Numbers {
				_, ok := expectedRes[v]
				rq.True(ok)
			}
		}
	})
}

func TestProcessUrls(t *testing.T) {
	t.Run("test result is sorted", func(t *testing.T) {
		rq := require.New(t)

		respPayload1 := payload{Numbers: []int{1, 2, 7}}
		respPayload2 := payload{Numbers: []int{1, 5}}

		server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			err := json.NewEncoder(w).Encode(respPayload1)

			rq.NoError(err)
		}))

		defer server1.Close()

		server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			err := json.NewEncoder(w).Encode(respPayload2)

			rq.NoError(err)
		}))

		defer server2.Close()

		results := processURLs(context.Background(), []string{server1.URL, server2.URL})
		finalResult := processFinalResult(results)

		rq.Equal([]int{1, 2, 5, 7}, finalResult)
	})
}

func TestNumbersHandler(t *testing.T) {
	url1 := "http://127.0.0.1:8090/primes"
	url2 := "http://127.0.0.1:8090/fibo"

	tt := []struct {
		name       string
		method     string
		path       string
		want       string
		statusCode int
		expectErr  bool
	}{
		{
			name:       "ok",
			method:     http.MethodGet,
			want:       `{"numbers": []}`,
			statusCode: http.StatusOK,
		},
		{
			name:       "error invalid method",
			method:     http.MethodPost,
			want:       `{"numbers": []}`,
			statusCode: http.StatusNotFound,
			expectErr:  true,
		},
		{
			name:       "error invalid path",
			method:     http.MethodGet,
			path:       "/nums",
			want:       `{"numbers": []}`,
			statusCode: http.StatusNotFound,
			expectErr:  true,
		},
	}

	for _, tc := range tt {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			rq := require.New(t)

			path := "/numbers"
			if tc.path != "" {
				path = tc.path
			}

			target := fmt.Sprintf("%s?u=%s&u=%s", path, url1, url2)

			request := httptest.NewRequest(tc.method, target, nil)
			response := httptest.NewRecorder()

			numbersHandler(response, request)

			rq.Equal(tc.statusCode, response.Code)

			if !tc.expectErr {
				rq.JSONEq(tc.want, response.Body.String())
			}
		})
	}
}
