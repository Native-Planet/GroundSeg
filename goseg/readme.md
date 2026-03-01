## GroundSeg API Golang rewrite (`goseg`)

```mermaid
stateDiagram-v2
    direction TB
    accTitle: groundseg package diagram
    accDescr: Interactions between packages in Groundseg Go rewrite
    classDef bcase fill:#f00,color:white,font-weight:bold,stroke-width:2px,stroke:yellow
    Broadcast-->WS_mux: broadcast latest update
    Broadcast-->Urbit_traffic: broadcast latest update
    Static-->Operations: imported
    Static-->Routines: imported
    state Internals {
        state Static {
            Structs
            Defaults
            Logger
        }
        state Routines {
            Docker_rectifier
            Version_server_fetch
            Startram_fetch
            Startram_rectifier
            Linux_updater
            502_refresher
            System_info
            mDNS
        }
        state Operations {
            Startram_config
            Docker
            System
            Config
            Transition
        }
        state Process_handler {
            WS_handler
            Startram_handler
            Support_handler
            Urbit_handler
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
    state External {
        Version_server
        Dockerhub
    }
    state Startram {
        StarTram_API
        WG_Server
    }
    External-->Routines: retrieve updated information
    Operations-->Startram: configure StarTram
    Routines-->Startram: retrieve remote config for startram
    state Docker_daemon {
        Urbit
        Minio
        MinioMC
        Netdata
        WireGuard
    }
    Operations-->Docker_daemon: manage containers
    Docker_daemon-->Startram: forward webui and ames
```

## Local Verification

Run before opening backend/runtime PRs:

1. `go test ./...`
2. `go test -tags=integration ./broadcast ./handler ./routines ./ws` (requires Docker and runtime dependencies)

## Websocket Action Contract (v1.0)

The shared websocket action registry now lives in `protocol/actions` and should be treated as the
single source of truth for command tokens that originate from the client:

- `c2c` domain (`goseg/system/wifi_*`)
- supported action: `connect` (`groundseg/protocol/actions.ActionC2CConnect`)
- `upload` domain (`goseg/uploadsvc`)
  - supported actions: `open-endpoint`, `reset`

Each domain keeps parse/dispatch parity by consuming:

- shared constants from `groundseg/protocol/actions`
- domain-specific parser wrappers (`ParseC2CAction`, `ParseAction`)
- shared unsupported action error semantics (`UnsupportedActionError`)

Compatibility guidance:

- Additions or removals to action surfaces require updates in `protocol/actions`
  and the domain parser/executor tests.
- Unknown actions must always return the shared unsupported action error contract.

When changing API boundaries, prefer explicit fetch vs sync naming (for example `Fetch*` read-only and `Sync*` mutating/persisting).
