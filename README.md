# Native Planet GroundSeg

![Groundseg 2 demo](https://user-images.githubusercontent.com/16911914/271397025-f534f6e3-5c62-4b9a-8cb9-9f1d07e5c29a.gif)

#### See the user manual [here](https://manual.groundseg.app) for usage documentation

GroundSeg is a software tool that helps users manage and access their multiple Urbit instances. 
It simplifies the process of getting onto the Urbit network and provides a range of additional services 
that enhance the functionality of the user's ship. With a [StarTram](https://www.nativeplanet.io/startram) 
subscription, users can also access their Urbit ship remotely.

Native Planet develops GroundSeg to run on our [dedicated Urbit-hosting home devices](https://www.nativeplanet.io)! 

## Dependencies

- `docker`
- `systemd`

### Semi-dependencies

These are optional external packages for used wifi support:

- `hostapd`
- `nmcli`

## Installation

**Disclaimer:** GroundSeg runs with `sudo` privileges on your device. This is required for controlling various aspects of the device. For this reason, we recommend a dedicated device.

### Docker + GroundSeg (Recommended)
This installs `docker` and the GroundSeg binary. Use this if you do not know what you're doing.

```
sudo wget -O - get.groundseg.app | bash
```

### Groundseg Only

This downloads the appropriate service file for you init system and the groundseg binary. Docker has to already be installed.

```
sudo wget -O - only.groundseg.app | bash
```

### Switching to the `edge` release channel (Unstable)

1. In `/opt/nativeplanet/groundseg/settings/system.json`, set `"updateBranch"` to `"edge"`
2. `sudo systemctl restart groundseg`

## Building From Source

1. Have docker installed
2. run `build.sh`

## Frontend Source And Runtime Ownership

- Source of truth for frontend runtime code is `ui/src/`.
- Generated web artifacts are output to `goseg/web/` by the UI build process.
- Do not hand-edit generated files under `goseg/web/_app/immutable/`; update `ui/src/` and rebuild.

Architecture and package-boundary details are documented in
`docs/architecture/frontend-runtime.md`.

## Frontend Quality Gate

Required checks before shipping frontend/runtime changes:

1. `cd ui && npm ci`
2. `npm run test`
3. `npm run build`

Only refresh committed `goseg/web/` artifacts after all checks pass.

## Backend Quality Gate

Required checks before shipping Go backend/runtime changes:

1. `cd goseg`
2. `go test ./...`
3. `go test -tags=integration ./broadcast ./handler ./routines ./ws` (run only in environments with Docker + runtime dependencies available)

Runtime boundary expectations for backend changes:

1. Preserve explicit error propagation (`%w`) at handler/service boundaries.
2. Keep fetch-only APIs separate from state-mutating sync APIs (for example `Fetch*` vs `Sync*`).
3. For flows that touch registration/version/upload paths, include at least one deterministic unit test in the changed package.
4. For upload and Wi-Fi websocket command contracts, follow package governance documents:
   - `goseg/uploadsvc/GOVERNANCE.md`
   - `goseg/system/wifi/README.md`
5. For StarTram external API masking semantics, follow `goseg/startram/GOVERNANCE.md`.
6. For protocol and error contract surfaces, define compatibility descriptors through:
   - `goseg/protocol/contracts/contracts.go` (`ContractDescriptor`, shared registry, active/deprecated helpers)
   - `goseg/protocol/contracts/protocol_contracts.go` (upload/C2C contract metadata and namespace bindings)
   - `goseg/protocol/contracts/startram_contracts.go` (StarTram error contract metadata)
   - `goseg/protocol/contracts/contracts_test.go` (explicit per-contract lifecycle-policy assertions for changed catalog entries)
   - `goseg/protocol/actions/actions.go` (action token contract adapters over registry descriptors)
   - `goseg/startram/errors.go` (API connection contract accessor and masking semantics)
   - Use the typed contract APIs (`ContractID`, `ActionNamespace`, `ActionToken`, and `ActionContractBinding`) instead of raw string namespace/action lookups.
7. Use shared boundary helpers for edge contracts:
   dependency-injected handlers (no package-global service mutation) and shared masked-error wrappers (`goseg/errpolicy`) for outward error semantics.

CI policy checks:

1. `Runtime Contract Gate` (`.github/workflows/upload-contract-gate.yml`) runs a path-aware check that fails if contract surfaces change without paired tests:
   `goseg/handler/ws/upload.go`, `goseg/uploadsvc/service.go`, `goseg/startram/errors.go`, `goseg/protocol/contracts/contracts.go`, `goseg/protocol/contracts/protocol_contracts.go`, `goseg/protocol/contracts/startram_contracts.go`, `goseg/protocol/actions/actions.go`.
2. The same gate runs targeted contract tests via `.github/scripts/check-upload-contract.sh` for upload branch matrix, upload dispatch parity, and startram masked-error semantics.

## Removing GroundSeg (Uninstall)

### Standard Removal (Recommended)
This removes `docker`, `docker-compose`, GroundSeg related docker containers and images, and the GroundSeg system files.
This **DOES NOT** remove the docker volumes on the device.

```
mkdir -p /tmp/nativeplanet && \
sudo wget -O /tmp/nativeplanet/standard_uninstall.sh \
https://raw.githubusercontent.com/Native-Planet/GroundSeg/master/release/standard_uninstall.sh && \
sudo chmod +x /tmp/nativeplanet/standard_uninstall.sh && \
sudo /tmp/nativeplanet/standard_uninstall.sh
```

### Groundseg Only

This removes GroundSeg related docker containers and images, and the GroundSeg system files.

```
mkdir -p /tmp/nativeplanet && \
sudo wget -O /tmp/nativeplanet/groundseg_uninstall.sh \
https://raw.githubusercontent.com/Native-Planet/GroundSeg/master/release/groundseg_uninstall.sh && \
sudo chmod +x /tmp/nativeplanet/groundseg_uninstall.sh && \
sudo /tmp/nativeplanet/groundseg_uninstall.sh
```

### Uninstall and clear data
This removes `docker`, `docker-compose`, **ALL** docker images, containers and volumes, and the GroundSeg system files.
This wipes all docker and GroundSeg data. Make sure you have exported the data you want saved.

```
mkdir -p /tmp/nativeplanet && \
sudo wget -O /tmp/nativeplanet/complete_uninstall.sh \
https://raw.githubusercontent.com/Native-Planet/GroundSeg/master/release/complete_uninstall.sh && \
sudo chmod +x /tmp/nativeplanet/complete_uninstall.sh && \
sudo /tmp/nativeplanet/complete_uninstall.sh
```
