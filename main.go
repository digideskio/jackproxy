package main

import (
  // "io"
  "io/ioutil"
  "net/http"
)


// func getWithRetry(url) {
//   var netTransport = &http.Transport{
//     Dial: (&net.Dialer{
//       Timeout: 15 * time.Second,
//     }).Dial,
//     TLSHandshakeTimeout: 15 * time.Second,
//   }
//   var netClient = &http.Client{
//     Timeout: time.Second * 40,
//     Transport: netTransport,
//   }
//   response, err := netClient.Get(url)
//   return response, err
// }

func hello(response http.ResponseWriter, request *http.Request) {
  resp, err := http.Get("http://example.com/")
  if err == nil {
    response.WriteHeader(http.StatusInternalServerError)
    return
  }
  defer resp.Body.Close()
  body, err := ioutil.ReadAll(resp.Body)
  response.Write(body)
}

func main() {
  http.HandleFunc("/", hello)
  http.ListenAndServe("127.0.0.1:8000", nil)
}
