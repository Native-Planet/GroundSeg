package system

import (
	"fmt"
	"net/http"

	"github.com/hsanjuan/go-captive"
	"groundseg/system/wifi/transport"
)

type captiveTransportAdapter struct {
	inner             *transport.CaptiveTransportAdapter
	processC2CMessage func([]byte) error
}

func newCaptiveTransportAdapter(deps c2cServiceDeps) *captiveTransportAdapter {
	adapter := &captiveTransportAdapter{
		processC2CMessage: func(msg []byte) error {
			return processC2CMessageForAdapterWithDeps(msg, deps)
		},
	}
	listSSIDs := func(dev string) ([]string, error) {
		return NewWiFiRuntimeService().ListWifiSSIDs(dev)
	}
	adapter.inner = transport.NewCaptiveTransportAdapter(listSSIDs, adapter.processMessage)
	return adapter
}

func (a *captiveTransportAdapter) runPortal(portal *captive.Portal) error {
	return a.inner.RunPortal(portal)
}

func (a *captiveTransportAdapter) handleAPI(w http.ResponseWriter, r *http.Request) {
	a.inner.HandleAPI(w, r)
}

func (a *captiveTransportAdapter) processMessage(msg []byte) error {
	if a.processC2CMessage == nil {
		return fmt.Errorf("no c2c message processor configured")
	}
	return a.processC2CMessage(msg)
}

func (a *captiveTransportAdapter) broadcastNetworks(dev string) {
	a.inner.BroadcastNetworks(dev)
}
