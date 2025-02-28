package core

import (
	"crypto/tls"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
)

// StaticServer represents a static file server
type StaticServer struct {
	// BindAddr is the address to bind the static server to
	// ex: ":8080"
	BindAddr string `json:"bindAddr"`
	// StaticDir is the directory to serve static files from
	// ex: "./static"
	StaticDir string `json:"staticDir"`
}

// RunStatic starts a static file server
func (c *Config) RunStatic() {

	// use the new OpenRoot because why not
	root, err := os.OpenRoot(c.StaticServer.StaticDir)
	if err != nil {
		slog.Error("static-server", "error", err)
		return
	}

	// extract the filesystem from the root
	fileSystem := root.FS()

	// create a new http server
	// TODO: maybe create a custom handler for static files to log requests
	server := &http.Server{
		Addr:    c.StaticServer.BindAddr,
		Handler: http.FileServerFS(fileSystem),
	}

	// both cert and key are needed, warn the user if they are not set
	if c.TLSCertFile != "" && c.TLSKeyFile != "" {
		slog.Info("static-server start tls", "cert", c.TLSCertFile, "key", c.TLSKeyFile, "bindAddr", server.Addr, "staticDir", c.StaticServer.StaticDir)
		err := server.ListenAndServeTLS(c.TLSCertFile, c.TLSKeyFile)
		if err != nil {
			slog.Error("static-server tls", "error", err)
			return
		}
		// otherwise, start the server without TLS
	} else {
		slog.Info("static-server start", "bindAddr", server.Addr, "staticDir", c.StaticServer.StaticDir)
		err := server.ListenAndServe()
		if err != nil {
			slog.Error("static-server", "error", err)
			return
		}
	}
	slog.Info("static-server shutdown")

}

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
func (c *Config) RunProxy() {

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
					slog.Debug("reverse-proxy add header", "key", k, "value", v)
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
		Addr:    c.BindAddr,
		Handler: mux,
	}

	slog.Info("reverse-proxy listen", "addr", server.Addr)

	// both cert and key are needed, warn the user if they are not set
	if c.TLSCertFile == "" && c.TLSKeyFile != "" {
		slog.Warn("reverse-proxy tls", "cert", "not set", "key", c.TLSKeyFile)
	} else if c.TLSCertFile != "" && c.TLSKeyFile == "" {
		slog.Warn("reverse-proxy tls", "cert", c.TLSCertFile, "key", "not set")
	}

	// if both cert and key are set, start the server with TLS
	if c.TLSCertFile != "" && c.TLSKeyFile != "" {
		slog.Info("reverse-proxy tls", "cert", c.TLSCertFile, "key", c.TLSKeyFile)
		err := server.ListenAndServeTLS(c.TLSCertFile, c.TLSKeyFile)
		if err != nil {
			slog.Error("reverse-proxy tls", "error", err)
			return
		}
		// otherwise, start the server without TLS
	} else {
		err := server.ListenAndServe()
		if err != nil {
			slog.Error("reverse-proxy", "error", err)
			return
		}
	}
	slog.Info("reverse-proxy shutdown")

}
