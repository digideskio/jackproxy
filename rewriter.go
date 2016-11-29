package main

import (
	"github.com/vulcand/oxy/forward"
	"github.com/vulcand/oxy/utils"
	"net/http"
)

type customRewriter struct {
}

func (rw *customRewriter) Rewrite(req *http.Request) {
	// TODO: this is a hack for making oxy copy the request correctly when the path changes.
	req.URL.Opaque = ""

	// Remove hop-by-hop headers to the backend.  Especially important is "Connection" because we
	// want a persistent connection, regardless of what the client sent to us.
	utils.RemoveHeaders(req.Header, forward.HopHeaders...)
}
