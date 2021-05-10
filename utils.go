package gowebsocket

import (
	"log"
	"net/http"
	"net/url"
)

// BuildProxy creates an http proxy
func BuildProxy(rawURL string) func(*http.Request) (*url.URL, error) {
	uProxy, err := url.Parse(rawURL)
	if err != nil {
		log.Fatal("Error while parsing url ", err)
	}
	return http.ProxyURL(uProxy)
}
