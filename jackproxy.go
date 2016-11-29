package main

import (
	"flag"
	"fmt"
	"github.com/vulcand/oxy/forward"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

// Flags.
var portFlag = flag.Int("port", 8080, "")
var proxymapPathFlag = flag.String("proxymap", "", "JSON config file for mapping hijacked paths")
var proxymeHostnameFlag = flag.String("proxyme-hostname", "proxyme.local", "")

// Healthcheck /healthz endpoint for the proxy itself.
func healthzHandler(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
	io.WriteString(w, "ok")
}

func normalizeRequestUrl(req *http.Request) {
	// Strip off :80 from hostport to match the proxymap.
	req.URL.Host = justHostname(req.URL.Host)

	if _, ok := globalProxymap[req.URL.String()]; !ok {
		// Fallback to non-query param URL if it exists. This helps sources that use query params like
		// for cache busting and rely on a common static webserver behavior that ignores query params
		// and still serves the file by path. This behavior is safe to allow and helps support serving
		// captured resources for static websites.
		req.URL.RawQuery = ""
		req.URL.ForceQuery = false
	}
}

func proxyHandler(w http.ResponseWriter, req *http.Request) {
	// Explicitly only allow GET/HEAD/OPTIONS requests, that's all we should support here.
	if req.Method != http.MethodGet &&
		req.Method != http.MethodHead &&
		req.Method != http.MethodOptions {
		// TODO: log actual error.
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	// Transform the request to force some local requests to the correct proxied address.
	proxifyIfLocalRequest(req)

	shouldBeProxied := strings.HasPrefix(req.URL.String(), "http://"+*proxymeHostnameFlag)
	if shouldBeProxied {
		normalizeRequestUrl(req)
	}

	if proxyItem, ok := globalProxymap[req.URL.String()]; ok {
		// URL is in the proxy map, hijack the request.
		fmt.Println("Proxying", req.URL, "-->", proxyItem.URL)

		req.Header.Set(internalMimetypeOverrideHeader, proxyItem.Mimetype)

		newUrl, err := url.ParseRequestURI(proxyItem.URL)
		if err != nil {
			// TODO: log actual error.
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		req.URL = newUrl
	} else {
		// URL is NOT in the proxy map.
		if shouldBeProxied || isBlacklistedUrl(req.URL.String()) {
			fmt.Println("Serving intentional 404 for:", req.URL.String())
			// We got a request for a proxied resource, but it's not in the proxymap so we don't know
			// where the resource exists. Immediately serve 404, otherwise we will attempt to connect
			// to the non-existent proxy host.
			http.Error(w, "", http.StatusNotFound)
			return
		} else {
			fmt.Println("Allowing live URL:", req.URL.String())
		}
		// If here, URL is a live URL and is requested without hijacking.
	}

	fwd, _ := forward.New(
		forward.Rewriter(&customRewriter{}),
		forward.RoundTripper(&CustomRoundTripper{}),
	)
	fwd.ServeHTTP(w, req)
}

func run() error {
	flag.Parse()

	if err := setupGlobalProxymap(*proxymapPathFlag); err != nil {
		return err
	}

	http.HandleFunc("/healthz", healthzHandler)
	http.HandleFunc("/", proxyHandler)
	http.ListenAndServe("127.0.0.1:"+strconv.Itoa(*portFlag), nil)
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
