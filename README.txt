This file documents the http server that listens on port 8080 and endpoint /numbers. The server is used to process a set of URLs that returns a list of numbers, sort those numbers and return the result as a JSON list of sorted numbers.

Since it is specified that the server has to return a response within 500ms, I have chosen this  as the timeout for each incoming request. Because we want to finish processing in less than or equal to 500ms time, there is another timeout called “timeoutGetReq” set to 400ms, so that all URLs send back their response within this time frame and we still have 100ms to 
process the results and sort them.

There is also a cache implemented in the server that keeps track of the last successful result of each URL. If a URL fails to get back a response or times out, this cache will be used to get the last stored result.

Some of the main functions used in the server and their short description is as follows:

numbersHandler():
This is the handler of the http server that receives the request from clients, processes them and then returns the result as JSON output.

processURLs():
This function is the one responsible for getting response from each URL. This is done in an asynchronous way using Go routines and channels.

processFinalResult():
This function processes the result returned by all the URLs, sorts them and returns the final list.