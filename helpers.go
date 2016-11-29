package main

import (
	"net"
	"net/http"
	"strings"
)

// HTML that will be injected in the footer of all HTML pages.
const animationStopperHtml = `<style type="text/css">
*, *::before, *::after {
  -moz-transition: none !important;
  transition: none !important;
  -moz-animation: none !important;
  animation: none !important;
}
</style>`

// Transforms localhost requests into requests that will be reverse proxied based on the proxymap.
func proxifyIfLocalRequest(req *http.Request) {
	hostname, _, _ := net.SplitHostPort(req.URL.Host)
	if hostname == "localhost" || hostname == "127.0.0.1" {
		req.Host = *proxymeHostnameFlag
		req.URL.Host = *proxymeHostnameFlag
	}
}

// Handle inconvenient host/hostport combos and just return host.
func justHostname(s string) string {
	if strings.Contains(s, ":") {
		hostname, _, _ := net.SplitHostPort(s)
		return hostname
	} else {
		return s
	}
}

func injectHtmlFooter(html []byte, footerHtml string) []byte {
	htmlString := string(html[:])
	if strings.Contains(htmlString, "</body>") {
		return []byte(strings.Replace(htmlString, "</body>", footerHtml+"</body>", -1))
	}
	if strings.Contains(htmlString, "</html>") {
		return []byte(strings.Replace(htmlString, "</html>", footerHtml+"</html>", -1))
	}
	return html
}

func isBlacklistedUrl(url string) bool {
	prefixBlacklist := []string{
		"http://ciscobinary.openh264.org/",
		"http://player.vimeo.com/",
		"http://cdn.krxd.net/ctjs/controltag.js",
	}
	for _, prefix := range prefixBlacklist {
		if strings.HasPrefix(url, prefix) {
			return true
		}
	}
	return false
}
