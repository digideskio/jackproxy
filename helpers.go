package main

import (
  "net"
  "net/http"
  "strings"
)

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
