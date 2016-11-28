package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/vulcand/oxy/forward"
	"github.com/vulcand/oxy/utils"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
)

var portFlag = flag.Int("port", 8080, "")
var proxymapPathFlag = flag.String("proxymap", "", "")

type ProxymapItem struct {
	URL      string `json:"url"`
	Mimetype string `json:"mimetype"`
}

var globalProxymap map[string]ProxymapItem

func setupGlobalProxymap(path string) error {
	proxymapData, err := ioutil.ReadFile(*proxymapPathFlag)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(proxymapData, &globalProxymap); err != nil {
		return err
	}
	return nil
}

// Healthcheck /healthz endpoint for the proxy itself.
func healthzHandler(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
	io.WriteString(w, "ok")
}

type Rewriter struct {
}

func (rw *Rewriter) Rewrite(req *http.Request) {
	// TODO: this is a hack for making oxy copy the request correctly when the path changes.
	req.URL.Opaque = ""

	// Remove hop-by-hop headers to the backend.  Especially important is "Connection" because we
	// want a persistent connection, regardless of what the client sent to us.
	utils.RemoveHeaders(req.Header, forward.HopHeaders...)
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

	// Check if the requested URL is in the proxymap. If it is, hijack the request.
	if proxyItem, ok := globalProxymap[req.URL.String()]; ok {
		fmt.Println("Proxying", req.URL, "-->", proxyItem.URL)

		newUrl, err := url.ParseRequestURI(proxyItem.URL)
		if err != nil {
			// TODO: log actual error.
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		req.URL = newUrl
	}

	fwd, _ := forward.New(forward.Rewriter(&Rewriter{}))
	fwd.ServeHTTP(w, req)
}

func run() error {
	flag.Parse()

	err := setupGlobalProxymap(*proxymapPathFlag)
	if err != nil {
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
