// Package captive provides a TCP/IP proxy for HTTP/HTTPs traffic controlled
// by a user-defined captive portal (client-IP based).
//
// In order to use the portal, TCP traffic for ports 80 and 443 must be
// forwarded to the portal ports (usually this is done using "iptables").
//
// The proxy acts as a man-in-the-middle only letting through HTTP(s) traffic
// from IPs which have been allowed. Along with the proxy, captive.Portal runs
// HTTP servers to redirect and serve the portal website, which is
// user-defined. The portal server handles a login endpoint which allows
// clients to be whitelisted based on a user-provided function.
package captive

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/tcpproxy"
)

// Portal creates a tcp/ip proxy for HTTP and HTTPs traffic that can be
// launched with Run(). Clients whose IPs have not been allowed are
// redirected to the PortalDomain. Clients visiting the PortalDomain
// will be served the website contents in WebPath.
//
// HTTP POST requests to LoginPath are used to allow clients by executing
// the LoginHandler function. If successful, traffic from authorized clients
// is let through to its original destination. GET requests to login path
// can be used to determine if a user is whitelisted or not.
//
// The captive portal operates on the TCP/IP layer, thus the client IP is used
// for authentication (and not MAC address -yet).
type Portal struct {
	// The TCP port to capture or allow HTTP requests through. 8080 by
	// default.
	HTTPPort int
	// The TCP port to allow HTTPs requests through. 8081 by default.
	HTTPSPort int
	// If you want to serve the portal with HTTPS, set CertFile and KeyFile
	// to a valid path. The certificate must be valid for the PortalDomain.
	CertFile string
	KeyFile  string
	// The path on disk of the Portal website shown to unlogged users and
	// seemingly running on PortalDomain. This should be a folder with at
	// least an index.html in it and is served using http.FileServer.
	WebPath string
	// The path to handle users logins and potentially allow HTTP and HTTPs
	// traffic. POST requests will trigger the LoginHandler and return
	// either 202 (Accepted) or 401 (Unauthorized).
	// GET requests will return 204 for authorized clients, or 403
	// for the rest. This should not be "/".
	LoginPath string
	// The captive's portal domain. Users get redirected here and served
	// the contents from the WebRoot directory. This domain must exist and
	// be resolvable.  Otherwise browsers will not make requests to it.
	PortalDomain string
	// Additonal subdomains to handle (i.e. "www", "login"). Defaults
	// to ["www"].
	PortalSubdomains []string
	// Are allowed clients shown the portal at all or are they let
	// through to the actual real PortalDomain site.
	AllowedBypassPortal bool
	// A Handler for LoginPath. Returns true if the client's traffic
	// should be let through or false otherwise. Not setting this will
	// authenticate all clients that request it.
	LoginHandler func(loginReq *http.Request) bool
	// An optional path to run the user-provided CustomHandler. This
	// allows the Portal to implement any other server-side functionality.
	CustomHandlerPath string
	// An optional custom http.HandleFunc to be called for
	// requests to CustomHandlerPath
	CustomHandler func(w http.ResponseWriter, r *http.Request)

	loginTarget    tcpproxy.Target
	httpProxy      tcpproxy.Target
	httpsProxy     tcpproxy.Target
	portalListener *tcpproxy.TargetListener
	redirectTarget *tcpproxy.TargetListener
	proxy          *tcpproxy.Proxy

	https         bool
	portalDomains []string

	whitelist *whitelist
}

// make sure fields are in order and set defaults
func (p *Portal) setup() error {
	if p.PortalDomain == "" {
		return errors.New("PortalDomain unset")
	}

	if p.LoginPath == "" || p.LoginPath == "/" {
		return errors.New("LoginPath invalid")
	}

	if chp := p.CustomHandlerPath; chp == p.LoginPath || chp == "/" {
		return errors.New("CustomHandlerPath invalid")
	}

	if p.CertFile != "" {
		if _, err := os.Stat(p.CertFile); err != nil {
			return fmt.Errorf("Bad CertFile: %s", err)
		}
	}

	if p.KeyFile != "" {
		if _, err := os.Stat(p.KeyFile); err != nil {
			return fmt.Errorf("Bad KeyFile: %s", err)
		}
	}

	if p.CertFile != "" && p.KeyFile != "" {
		p.https = true
	}

	if p.HTTPPort == 0 {
		p.HTTPPort = 8080
	}

	if p.HTTPSPort == 0 {
		p.HTTPSPort = 8081
	}

	if p.PortalSubdomains == nil {
		p.PortalSubdomains = []string{"www"}
	}

	p.portalDomains = []string{p.PortalDomain}
	for _, sub := range p.PortalSubdomains {
		p.portalDomains = append(p.portalDomains, fmt.Sprintf("%s.%s", sub, p.PortalDomain))
	}

	if p.LoginHandler == nil {
		p.LoginHandler = func(r *http.Request) bool {
			return true
		}
	}

	p.portalListener = new(tcpproxy.TargetListener)
	p.loginTarget = p.portalListener
	p.redirectTarget = new(tcpproxy.TargetListener)
	p.proxy = new(tcpproxy.Proxy)
	p.whitelist = &whitelist{}

	// The httpProxy allowed proxy will receive traffic for all non-portal
	// domains and redirect non-allowed traffic to the portal.
	p.httpProxy = &allowedProxy{p.whitelist, 80, p.redirectTarget}
	// the httpsProxy will just allow https traffic through or close
	// the connection if the client is not whitelisted.
	p.httpsProxy = &allowedProxy{p.whitelist, 443, nil}

	// In this case, requests to the portal should be let through
	// to the non-hijecked destination. When the portal is https,
	// this is the only case when allowed handles non allowed https
	// traffic (to serve the portal)
	if p.AllowedBypassPortal {
		byPassPort := 80
		if p.https {
			byPassPort = 443
		}
		p.loginTarget = &allowedProxy{p.whitelist, byPassPort, p.portalListener}
	}
	return nil
}

func (p *Portal) portalServer() error {
	mux := http.NewServeMux()
	mux.HandleFunc(p.LoginPath, func(w http.ResponseWriter, r *http.Request) {
		ip := extractIP(r.RemoteAddr)
		log.Printf("Serving portal to %s\n", ip)

		if r.Method == "GET" {
			retVal := http.StatusForbidden
			if p.whitelist.isAllowed(ip) {
				retVal = http.StatusNoContent
			}
			w.WriteHeader(retVal)
			return
		}

		if allowed := p.LoginHandler(r); allowed {
			p.whitelist.add(ip)

			// Close connection so clients can
			// reload the portal domain  and see
			// the real.
			if p.AllowedBypassPortal {
				r.Close = true
			}
			w.WriteHeader(http.StatusAccepted)
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	})

	if p.CustomHandler != nil && p.CustomHandlerPath != "" {
		mux.HandleFunc(p.CustomHandlerPath, p.CustomHandler)
	}

	fs := http.FileServer(http.Dir(p.WebPath))
	noCache := &noCacheHandler{fs}
	mux.Handle("/", noCache)

	s := &http.Server{
		Handler:           mux,
		ReadTimeout:       5 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       500 * time.Millisecond,
	}
	var err error
	if p.https {
		log.Println("Launching portal server (https)")
		err = s.ServeTLS(p.portalListener, p.CertFile, p.KeyFile)
	} else {
		log.Println("Launching portal server (http)")
		err = s.Serve(p.portalListener)
	}
	if err != nil {
		log.Println(err)
	}
	return err
}

// launches the redirect server (always on http)
func (p *Portal) redirectServer() error {
	scheme := "http"
	if p.https {
		scheme = "https"
	}
	redirectMux := http.NewServeMux()
	redirectMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, scheme+"://"+p.PortalDomain, http.StatusFound)
	})

	log.Println("Launching redirect server (http)")
	err := http.Serve(p.redirectTarget, redirectMux)
	if err != nil {
		log.Println(err)
	}
	return err
}

// Run starts the proxy and  captive portal servers. It will throw errors
// if some Portal fields are not correctly set.
func (p *Portal) Run() error {
	err := p.setup()
	if err != nil {
		return err
	}

	go p.portalServer()
	go p.redirectServer()

	httpPort := fmt.Sprintf(":%d", p.HTTPPort)
	httpsPort := fmt.Sprintf(":%d", p.HTTPSPort)

	// Requests for the portal domains get handled by the login target.
	for _, d := range p.portalDomains {
		if p.https {
			p.proxy.AddSNIRoute(httpsPort, d, p.loginTarget)
			p.proxy.AddHTTPHostRoute(httpPort, d, p.redirectTarget)
		} else {
			p.proxy.AddHTTPHostRoute(httpPort, d, p.loginTarget)
		}
	}

	// Requests for any other Host get proxied. httpProxy will
	// take care of redirecting to the portal domain when not allowed
	p.proxy.AddHTTPHostMatchRoute(httpPort, matchAll, p.httpProxy)
	p.proxy.AddSNIMatchRoute(httpsPort, matchAll, p.httpsProxy)

	scheme := "http"
	if p.https {
		scheme = "https"
	}

	log.Printf("Running TCP proxy:")
	log.Printf("  Captive portal domain: %s://%s", scheme, p.PortalDomain)
	log.Printf("  Additional subdomains: %s", p.PortalSubdomains)
	log.Printf("  Login handler path: POST %s", p.LoginPath)
	log.Printf("  Proxy ports: %d (HTTP), %d (HTTPS)", p.HTTPPort, p.HTTPSPort)
	return p.proxy.Run()
}

// Close shuts down the proxy and the portal servers.
func (p *Portal) Close() error {
	if t := p.redirectTarget; t != nil {
		t.Close()
	}
	if t := p.portalListener; t != nil {
		t.Close()
	}
	if t := p.proxy; t != nil {
		t.Close()
	}
	return nil
}

func extractIP(ipport string) string {
	parts := strings.Split(ipport, ":")
	last := len(parts) - 1
	return strings.Join(parts[0:last], ":")
}

func matchAll(ctx context.Context, hostname string) bool {
	return true
}

type noCacheHandler struct {
	h http.Handler
}

func (nch *noCacheHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	nch.h.ServeHTTP(w, r)
}

type whitelist struct {
	smap sync.Map
}

func (wl *whitelist) isAllowed(ip string) bool {
	_, ok := wl.smap.Load(ip)
	return ok
}

func (wl *whitelist) add(ip string) {
	wl.smap.Store(ip, nil)
}
