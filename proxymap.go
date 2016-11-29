package main

import (
	"encoding/json"
	"io/ioutil"
)

type ProxymapItem struct {
	URL      string `json:"url"`
	Mimetype string `json:"mimetype"`
}

var globalProxymap map[string]ProxymapItem

func setupGlobalProxymap(path string) error {
	proxymapData, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(proxymapData, &globalProxymap); err != nil {
		return err
	}
	return nil
}
