package main

import (
	"flag"
	"fmt"
	log "github.com/Sirupsen/logrus"
	logrus_syslog "github.com/Sirupsen/logrus/hooks/syslog"
	"github.com/vulcand/oxy/forward"
	"io"
	"log/syslog"
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
	portString := strconv.Itoa(*portFlag)

	// Explicitly only allow GET/HEAD/OPTIONS requests, that's all we should support here.
	if req.Method != http.MethodGet &&
		req.Method != http.MethodHead &&
		req.Method != http.MethodOptions {
		log.Error("[jackproxy][", portString, "] ", req.Method, " not allowed for: ", req.URL.String())
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	// Transform the request to force some local requests to the correct proxied address.
	originalUrl := req.URL.String()
	proxifyIfLocalRequest(req)

	shouldBeProxied := strings.HasPrefix(req.URL.String(), "http://"+*proxymeHostnameFlag)
	if shouldBeProxied {
		normalizeRequestUrl(req)
	}

	if proxyItem, ok := globalProxymap[req.URL.String()]; ok {
		// URL is in the proxy map, hijack the request.
		log.Info("[jackproxy][", portString, "] Proxying ", originalUrl, " --> ", proxyItem.URL)

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
			log.Info("[jackproxy][", portString, "] Serving intentional 404 for: ", originalUrl)
			// We got a request for a proxied resource, but it's not in the proxymap so we don't know
			// where the resource exists. Immediately serve 404, otherwise we will attempt to connect
			// to the non-existent proxy host.
			http.Error(w, "", http.StatusNotFound)
			return
		} else {
			log.Info("[jackproxy][", portString, "] Allowing live URL: ", req.URL.String())
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

	// Parse the proxymap and store it.
	if err := setupGlobalProxymap(*proxymapPathFlag); err != nil {
		return err
	}

	// Make sure the current proxyme hostname is marked as local, so it doesn't need any DNS lookups.
	markHostnamesLocal("localhost", "127.0.0.1", "testserver", *proxymeHostnameFlag)

	// Set up syslog.
	if hook, err := logrus_syslog.NewSyslogHook("", "", syslog.LOG_INFO, ""); err == nil {
		log.SetFormatter(&log.TextFormatter{DisableColors: true})
		log.AddHook(hook)
	} else {
		return err
	}

	// Run the proxy.
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
