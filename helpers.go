package main

import (
	"github.com/orcaman/concurrent-map"
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

// Use a concurrent map for the local hostname cache to avoid concurrent map access crashes.
var localHostnameCache = cmap.New()

func markHostnamesLocal(hostnames ...string) {
	for _, hostname := range hostnames {
		localHostnameCache.Set(hostname, true)
	}
}

// Bespoke, Percy-specific functionality for determining if a hostname is equivalent to localhost.
// This dynamically handles DOM snapshots from custom local server hostnames like "testserver" or
// "local.dev" where they should be proxyified here so that the asset is still loaded correctly
// in the rendering environment.
func isLocalHostname(hostname string) bool {
	// Don't do reverse DNS lookups on IP addresses, just assume they are not local.
	if hostname != "127.0.0.1" && net.ParseIP(hostname) != nil {
		return false
	}

	// Go's DNS timeouts for NX records are slow and the net package has no mechanism to configure
	// them. For speed, keep a cache of hostnames which we already consider local.
	if _, ok := localHostnameCache.Get(hostname); ok {
		return true
	}

	// Do a DNS lookup: if addresses at all are returned, don't consider the request "local".
	addrs, _ := net.LookupHost(hostname)
	if len(addrs) == 0 {
		markHostnamesLocal(hostname)

		// Assume that if a host cannot be looked up, we should fallback to local proxymap resolution.
		return true
	}
	return false
}

// Transforms localhost requests into requests that will be reverse proxied based on the proxymap.
func proxifyIfLocalRequest(req *http.Request) {
	hostname := justHostname(req.URL.Host)

	if isLocalHostname(hostname) {
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
		// NOTE: this blocks Firefox from downloading a hefty video codec in the background.
		"http://ciscobinary.openh264.org:80",
	}
	for _, prefix := range prefixBlacklist {
		if strings.HasPrefix(url, prefix) {
			return true
		}
	}
	return false
}
