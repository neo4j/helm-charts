package proxy

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
)

func httpProxy(hostname string) (*httputil.ReverseProxy, error) {
	url, err := url.Parse(fmt.Sprintf("http://%s:7474", hostname))
	if err != nil {
		return nil, err
	}
	proxy := httputil.NewSingleHostReverseProxy(url)

	// Modify response
	proxy.ModifyResponse = func(response *http.Response) error {

		if response.Header.Get("Content-Type") == "application/json" {
			bodyBytes, err := io.ReadAll(response.Body)
			if err != nil {
				return fmt.Errorf("error while reading json response \n %v", err)
			}
			portInt, err := strconv.Atoi(os.Getenv("PORT"))
			if err != nil {
				return err
			}
			//subtracting 8000 from the port number since we are adding 8000 in the helm chart template so as to not use port range < 1024
			portInt -= 8000
			port := fmt.Sprintf(":%d", portInt)
			b := bytes.Replace(bodyBytes, []byte(":7687"), []byte(port), -1)
			response.Header.Set("Content-Length", strconv.Itoa(len(b)))
			response.Body = io.NopCloser(bytes.NewReader(b))
		}
		return nil
	}

	return proxy, nil
}

func boltProxy(hostname string) (*httputil.ReverseProxy, error) {
	url, err := url.Parse(fmt.Sprintf("http://%s:7687", hostname))
	if err != nil {
		return nil, err
	}
	return httputil.NewSingleHostReverseProxy(url), nil
}
