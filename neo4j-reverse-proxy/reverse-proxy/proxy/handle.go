package proxy

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
)

type Handle struct {
	HostName   string
	BoltProxy  *httputil.ReverseProxy
	Neo4jProxy *httputil.ReverseProxy
}

func NewHandle() (*Handle, error) {
	hostname := fmt.Sprintf("%s.%s.svc.%s", os.Getenv("SERVICE_NAME"), os.Getenv("NAMESPACE"), os.Getenv("DOMAIN"))
	log.Printf("Hostname := %s", hostname)
	if ip, present := os.LookupEnv("IP"); present {
		hostname = ip
	}
	neo4jProxy, err := httpProxy(hostname)
	if err != nil {
		return nil, err
	}
	bProxy, err := boltProxy(hostname)
	if err != nil {
		return nil, err
	}
	return &Handle{
		HostName:   hostname,
		BoltProxy:  bProxy,
		Neo4jProxy: neo4jProxy,
	}, nil
}

func (h Handle) ServeHTTP(responseWriter http.ResponseWriter, request *http.Request) {

	proxy := h.Neo4jProxy
	if request.Header.Get("Upgrade") == "websocket" {
		proxy = h.BoltProxy
	}
	proxy.ServeHTTP(responseWriter, request)
}
