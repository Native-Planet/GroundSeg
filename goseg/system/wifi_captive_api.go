package system

import (
	"net/http"

	"github.com/hsanjuan/go-captive"
)

var (
	captiveAdapter = newCaptiveTransportAdapter(defaultC2CServiceDeps())
	captivePortal  = &captive.Portal{
		LoginPath:           "/",
		PortalDomain:        "nativeplanet.local",
		AllowedBypassPortal: false,
		WebPath:             "c2c",
	}
)

func CaptivePortal(_ string) error {
	return captiveAdapter.runPortal(captivePortal)
}

func CaptiveAPI(w http.ResponseWriter, r *http.Request) {
	captiveAdapter.handleAPI(w, r)
}

func announceNetworks(device string) {
	captiveAdapter.broadcastNetworks(device)
}
