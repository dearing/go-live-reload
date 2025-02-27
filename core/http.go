package core

import (
	"crypto/tls"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

// HttpTarget is a reverse proxy target
type HttpTarget struct {

	// Host is the URL of the target
	// ex: "http://localhost:8080"
	Host string `json:"host"`

	// CustomHeaders is a map of headers to add to the request
	// ex: {"Speak-Friend": "mellon"}
	CustomHeaders map[string]string `json:"customHeaders,omitzero"`

	// InsecureSkipVerify is a flag to enable or disable TLS verification downstream
	InsecureSkipVerify bool `json:"insecureSkipVerify,omitzero"`
}

// RunProxy starts a reverse proxy server
func (c *Config) RunProxy(addr string) {

	slog.Info("reverse-proxy init")

	mux := http.NewServeMux()

	// add each reverse proxy target to our MIX
	for path, target := range c.ReverseProxy {

		// parse the target into a URL (scheme, host, port)
		url, err := url.Parse(target.Host)
		if err != nil {
			slog.Error("reverse-proxy", "error", err, "target", target)
			return
		}

		// create a new reverse proxy
		proxy := &httputil.ReverseProxy{

			// ErrorHandler is a function that is called when the reverse proxy encounters an error
			ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
				slog.Error("reverse-proxy", "path", path, "host", target.Host, "error", err)
				http.Error(w, err.Error(), http.StatusBadGateway)
			},

			// Director is an (oddly named) function that modifies the request before it is sent
			Director: func(r *http.Request) {

				// add any custom headers to the request
				for k, v := range target.CustomHeaders {
					slog.Debug("reverse-proxy header", "key", k, "value", v)
					r.Header.Add(k, v)
				}

				incoming := r.URL.Path

				// TODO: this still feels too clunky, selectively manipulating the request
				r.URL.Scheme = url.Scheme
				r.URL.Host = url.Host
				r.URL.Path = strings.TrimPrefix(incoming, "/api")

				if !strings.HasPrefix(r.URL.Path, "/") {
					r.URL.Path = "/" + r.URL.Path
				}

				slog.Info("reverse-proxy", "path", path, "host", target.Host, "incoming", incoming, "downstream", r.URL.Path)

			},
		}

		// set the transport to allow insecure connections
		proxy.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: target.InsecureSkipVerify,
			},
		}
		mux.Handle(path, proxy)
		slog.Info("reverse-proxy handle", "path", path, "host", target.Host)
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
