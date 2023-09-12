## GroundSeg API Golang rewrite (`goseg`)

```mermaid
stateDiagram-v2
    direction TB
    accTitle: Goseg package diagram
    accDescr: Interactions between packages in Groundseg Go rewrite
    Broadcast-->WS_mux: broadcast latest update
    Broadcast-->Urbit_traffic: broadcast latest update
    Static-->Operations: imported
    Static-->Routines: imported
    state Internals {
        state Static {
            Structs
            Defaults
        }
        state Routines {
            Docker_rectifier
            Version_server_fetch
            Startram_fetch
            Startram_rectifier
            Linux_updater
            502_refresher
        }
        state Operations {
            Startram
            Docker
            System
            Config
            Transition
        }
        Process_handler-->Operations: multiple function calls to these packages to string together actions
        Operations-->Broadcast: send updated values
        Routines-->Broadcast: send updated values
    }
    [*]-->WS_mux
    [*]-->Urbit_traffic
    WS_mux-->Process_handler: valid request
    Urbit_traffic-->Process_handler: valid request
    Routines-->Operations: same as process handler
    state interfaces {
        state Urbit_traffic {
            UrbitAuth-->Lick
            Lick-->UrbitAuth
        }
        state WS_mux {
            WsAuth-->Websocket: broadcast structure out
            Websocket-->WsAuth: action payload in
   }
}
```