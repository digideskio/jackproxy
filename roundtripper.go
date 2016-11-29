package main

import (
  "bytes"
  "fmt"
  "net/http"
  "io/ioutil"
  "strings"
  "strconv"
)

// An internal-only header that is used to pass information on the request into this round-tripper.
// TODO: what is a more go idiomatic way to store this custom data on the request?
var internalMimetypeOverrideHeader = "X-Temp-JackProxy-Response-Content-Type"

// Number of retries for 5XX errors.
var numRetries = 3

// HTML that will be injected in the footer of all HTML pages.
const animationStopperHtml = `<style type="text/css">
*, *::before, *::after {
  -moz-transition: none !important;
  transition: none !important;
  -moz-animation: none !important;
  animation: none !important;
}
</style>`

// Custom transporter that provides response rewriting before headers are flushed by Forwarder.
type CustomRoundTripper struct{}

func (t *CustomRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
  // Extract the header override before actually making the request. This way, we don't
  // ever expose it to external services.
  mimetypeOverride := req.Header.Get(internalMimetypeOverrideHeader)
  req.Header.Del(internalMimetypeOverrideHeader)

  response, err := t.RoundTripWithRetries(req)
  if err != nil {
    return response, err
  }

  // Override the mimetype.
  if mimetypeOverride != "" {
    response.Header.Set("Content-Type", mimetypeOverride)
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

// RoundTrip method that retries server errors a few times.
func (t *CustomRoundTripper) RoundTripWithRetries(req *http.Request) (*http.Response, error) {
  var response *http.Response
  var err error

  for i := 0; i <= numRetries; i++ {
    response, err = http.DefaultTransport.RoundTrip(req)
    if err != nil {
      return response, err
    }
    if response.StatusCode < 500 {
      break
    }
    fmt.Println("Retrying", response.StatusCode, "response: ", req.URL.String())
  }
  return response, err
}

func injectHtmlFooter(html []byte, footerHtml string) []byte {
  htmlString := string(html[:])
  if strings.Contains(htmlString, "</body>") {
    return []byte(strings.Replace(htmlString, "</body>", footerHtml + "</body>", -1))
  }
  if strings.Contains(htmlString, "</html>") {
    return []byte(strings.Replace(htmlString, "</html>", footerHtml + "</html>", -1))
  }
  return html
}