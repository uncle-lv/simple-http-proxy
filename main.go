package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"

	"github.com/uncle-lv/logger"
)

var hopByHopHeaders = []string{
	"Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"TE",
	"Trailers",
	"Transfer-Encoding",
	"Upgrade",
}

type proxy struct {
}

func (p *proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger.Info(r.RemoteAddr, " ", r.Method, " ", r.URL)

	if r.URL.Scheme != "http" && r.URL.Scheme != "https" {
		msg := "unsupport protocol scheme: " + r.URL.Scheme
		http.Error(w, msg, http.StatusBadRequest)
		logger.Error(msg)
		return
	}

	r.RequestURI = ""
	removeHopByHopHeaders(r.Header)
	appendForwardedHeader(r)
	logger.Debug(r.Header["X-Forwarded-For"])

	client := &http.Client{}
	resp, err := client.Do(r)
	if err != nil {
		http.Error(w, "Server Error", http.StatusInternalServerError)
		logger.Fatal(err)
	}
	defer resp.Body.Close()

	removeHopByHopHeaders(resp.Header)
	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func removeHopByHopHeaders(header http.Header) {
	for _, h := range hopByHopHeaders {
		header.Del(h)
	}
}

func appendForwardedHeader(r *http.Request) {
	clientIP, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		logger.Error(err)
		return
	}

	if _, ok := r.Header["X-Forwarded-For"]; ok {
		r.Header.Add("X-Forwarded-For", clientIP)
	} else {
		r.Header.Set("X-Forwarded-For", clientIP)
	}
}

func copyHeader(dst, src http.Header) {
	for name, paras := range src {
		for _, para := range paras {
			dst.Add(name, para)
		}
	}
}

func main() {
	port := flag.Int("port", 8080, "The port of the proxy server")
	flag.Parse()
	p := &proxy{}

	logger.Info("Starting proxy server on port: ", *port)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", *port), p); err != nil {
		logger.Fatal(err)
	}
}
