package captive

import (
	"fmt"
	"log"
	"net"

	"github.com/google/tcpproxy"
)

// allowedProxy redirects non allowed TCP-HTTP(s) requests to
// a custom target while letting allowed requests through to their
// original destination
type allowedProxy struct {
	whitelist        *whitelist
	port             int
	nonAllowedTarget *tcpproxy.TargetListener
}

func (pxy *allowedProxy) HandleConn(c net.Conn) {
	tconn, ok := c.(*tcpproxy.Conn)
	if !ok { // non proxied traffic gets dropped
		return
	}

	ip := extractIP(c.RemoteAddr().String())
	if pxy.whitelist.isAllowed(ip) {
		if hn := tconn.HostName; hn != "" {
			// proxy this connection
			log.Printf("Redirect: Proxying allowed to %s:%d (%s)", hn, pxy.port, ip)
			dp := tcpproxy.To(fmt.Sprintf("%s:%d", tconn.HostName, pxy.port))
			dp.HandleConn(c)
			return
		}

		// Hostname not set, do not proxy.
		c.Close()
		return
	}

	log.Printf("Proxying disallowed to %s:%d (%s)", tconn.HostName, pxy.port, ip)

	// We cannot redirect HTTPs nicely, so we just close the connection.
	if pxy.port == 443 && pxy.nonAllowedTarget == nil {
		c.Close()
		return
	}

	// For the rest, we can just use the nonAllowedTarget.
	pxy.nonAllowedTarget.HandleConn(c)
}
