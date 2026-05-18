package subtitle

import (
	"io"
	"net/http"
)

// newHTTPRequest creates a new HTTP request with the given method, URL and body.
func newHTTPRequest(method, url string, body io.Reader) (*http.Request, error) {
	return http.NewRequest(method, url, body)
}

// doRequest executes the request and returns the response.
func doRequest(req *http.Request) (*http.Response, error) {
	return http.DefaultClient.Do(req)
}