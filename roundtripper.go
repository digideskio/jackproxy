package main

import (
	"bytes"
	log "github.com/Sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"strconv"
)

// An internal-only header that is used to pass information on the request into this round-tripper.
// TODO: what is a more go idiomatic way to store this custom data on the request?
var internalMimetypeOverrideHeader = "X-Temp-JackProxy-Response-Content-Type"

// Number of retries for 5XX errors.
var numRetries = 3

// Custom transporter that provides response rewriting before headers are flushed by Forwarder.
type CustomRoundTripper struct{}

func (t *CustomRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Extract the header override before actually making the request. This way, we don't
	// ever expose it to external services.
	mimetypeOverride := req.Header.Get(internalMimetypeOverrideHeader)
	req.Header.Del(internalMimetypeOverrideHeader)
	isProxied := mimetypeOverride != ""

	response, err := t.RoundTripWithRetries(req)
	if err != nil {
		return response, err
	}
	if isProxied {
		// Override mimetype. This enables serving the same destination with different mimetypes
		// per request (imagine a file served as text/html in one request and text/plain in another).
		response.Header.Set("Content-Type", mimetypeOverride)

		// Per spec, some browsers require that font files are served with the correct Access Control
		// settings. For now, just set this on all proxied requests since the rendering environment
		// is a controlled place (note: you would never want to do this in a normal proxy).
		response.Header.Set("Access-Control-Allow-Origin", "*")
	}

	// Inject custom HTML.
	if response.Header.Get("Content-Type") == "text/html" {
		html, _ := ioutil.ReadAll(response.Body)
		newHtml := injectHtmlFooter(html, animationStopperHtml)

		// Since we have already read the body, wrap it with NopCloser so it can be read again.
		response.Body = ioutil.NopCloser(bytes.NewBuffer(newHtml))
		response.ContentLength = int64(len(newHtml))
		response.Header.Set("Content-Length", strconv.Itoa(len(newHtml)))
	}

	return response, err
}

func (t *CustomRoundTripper) RoundTripWithRetries(req *http.Request) (*http.Response, error) {
	portString := strconv.Itoa(*portFlag)

	var response *http.Response
	var err error

	for i := 0; i <= numRetries; i++ {
		response, err = http.DefaultTransport.RoundTrip(req)
		if err == nil && response.StatusCode < 500 {
			// Got a correct response (non-5XX).
			break
		} else if err == nil {
			// Error handling: connected to host, but got 5XX responses.
			log.Info("[jackproxy][", portString, "] Retrying, got ", response.StatusCode, " for: ", req.URL.String())
		} else {
			// Error handling for connections, errors like: dial tcp: lookup example.com: no such host.
			// Returns a 502 Bad Gateway response if retries don't work.
			log.Info("[jackproxy][", portString, "] Retrying, got error (", err, ") for: ", req.URL.String())
		}
	}
	return response, err
}
