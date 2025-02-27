package core

import (
	"crypto/tls"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

type HttpTarget struct {
	Path               string `json:"path"`
	Host               string `json:"host"`
	StripPrefix        bool   `json:"stripPrefix"`
	InsecureSkipVerify bool   `json:"insecureSkipVerify"`
}

// RunProxy starts a reverse proxy server
func (c *Config) RunProxy(addr string) {

	slog.Info("reverse-proxy init")

	mux := http.NewServeMux()

	// add each reverse proxy target to our MIX
	for _, target := range c.ReverseProxy {

		// parse the target into a URL (scheme, host, port)
		url, err := url.Parse(target.Host)
		if err != nil {
			slog.Error("reverse-proxy", "error", err, "target", target)
			return
		}

		// create a new reverse proxy
		proxy := &httputil.ReverseProxy{

			// Director is an (oddly named) function that modifies the request before it is sent
			Director: func(r *http.Request) {

				incoming := r.URL.Path

				// TODO: this still feels too clunky, selectively manipulating the request
				r.URL.Scheme = url.Scheme
				r.URL.Host = url.Host
				r.URL.Path = strings.TrimPrefix(incoming, "/api")

				if !strings.HasPrefix(r.URL.Path, "/") {
					r.URL.Path = "/" + r.URL.Path
				}

				slog.Info("reverse-proxy", "path", target.Path, "host", target.Host, "incoming", incoming, "downstream", r.URL.Path)

			},
		}

		// set the transport to allow insecure connections
		proxy.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: target.InsecureSkipVerify,
			},
		}

		mux.Handle(target.Path, proxy)

		slog.Info("reverse-proxy handle", "path", target.Path, "host", target.Host)
	}

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	slog.Info("reverse-proxy listen", "addr", server.Addr)

	err := server.ListenAndServe()
	if err != nil {
		slog.Error("reverse-proxy", "error", err)
	}

	slog.Info("reverse-proxy shutdown")

}
