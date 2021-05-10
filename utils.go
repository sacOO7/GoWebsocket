package gowebsocket

import (
	"log"
	"net/http"
	"net/url"
)

func BuildProxy(Url string) func(*http.Request) (*url.URL, error) {
	uProxy, err := url.Parse(Url)
	if err != nil {
		log.Fatal("Error while parsing url ", err)
	}
	return http.ProxyURL(uProxy)
}
