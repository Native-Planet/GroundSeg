# go-captive

A very simple library to build captive portals.

`go-captive` handles the whitelisting, redirection and forwarding of HTTP(s) traffic to a user-defined captive portal. It provides the captive portal developer with the necessary shortcuts to just setup a custom "login" handler to allow/deny client's access to the internet.

The portal works as a man-in-the-middle TCP proxy. Non-allowed clients are forwarded to a handler which triggers an HTTP redirect to the user-provided portal server. This means your application needs to be configured as a traffic proxy, receiving all the traffic of the clients (see below).

Modern software like Firefox or Android should automatically detect a captive-portal and offer the user to redirect to it. Otherwise, any unallowed HTTP traffic will be redirected to the portal. Unallowed HTTPs is terminated.

## Usage

See https://godoc.org/github.com/hsanjuan/go-captive for library documentation.

You will need to provide your own Captive Portal Website (TODO: provide an
example one), and instantiate `captive.Portal` as part of your Go application.

Simplest usage:

```go
package main


import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/hsanjuan/go-captive"
)

func loginHandler(r *http.Request) bool {
	// Clicking on a "/login" link will allow traffic for that user.
	return true
}

func main() {
	proxy := &captive.Portal{
		LoginPath:           "/login",
		PortalDomain:        "myCaptivePortalDomain.com",
		AllowedBypassPortal: false,
		WebPath:             "staticContentFolder",
		LoginHandler:        loginHandler,
	}

	// For local debugging, run the website also on a local port.
	go func() {
		http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
			ok := loginHandler(r)
			if ok {
				w.WriteHeader(http.StatusAccepted)
			} else {
				w.WriteHeader(http.StatusUnauthorized)
			}
		})
		fs := http.FileServer(http.Dir("staticContentFolder"))
		http.Handle("/", fs)
		log.Fatal(http.ListenAndServe(":9080", nil))
	}()

	err := proxy.Run()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
```

## Proxy configuration

If you want to run a Wifi hotspot, you can use `hostapd` and
[`create_ap`](https://github.com/oblique/create_ap):

```sh
create_ap -d wlan1            wlan0          wlan-name <password>

             ^^hotspot-iface  ^^bridge-iface
```

Once you have an interface that is receiving all client traffic, you need to
forward it to the captive portal so it is allowed or denied. This is usually
done with `iptables`.

This assumes the interface receiving client traffic is `ap0` and that the
captive portal is running on ports tcp:8080 (http) and tcp:8081 (https).

Run as root:

```
iface=ap0
sysctl -w net.ipv4.ip_forward=1
sysctl -w net.ipv6.conf.all.forwarding=1
sysctl -w net.ipv4.conf.all.send_redirects=0

iptables -I INPUT -p tcp -m tcp --dport 8080 -j ACCEPT
iptables -I INPUT -p tcp -m tcp --dport 8081 -j ACCEPT
iptables -t nat -A PREROUTING -i $iface -p tcp --dport 80 -j REDIRECT --to-port 8080
iptables -t nat -A PREROUTING -i $iface -p tcp --dport 443 -j REDIRECT --to-port 8081

```
