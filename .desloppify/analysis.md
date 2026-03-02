Investigated #1: moved config event handling into explicit startable loop and removed implicit launch from config.Initialize. Added StartConfEventLoop(ctx, <-chan string) with cancel semantics, kept legacy ConfChannel wrapper. Started loop from startup runtime using context. Added tests for event processing/cancelation and fixed auth-session fake-client handling to keep token-id auth behavior for nil websocket connections.

Investigating issue 1: remaining singleton forwarder duplication remains in shipworkflow and startup wiring.
- In shipworkflow/operations_runtime.go, exported transition/lookup helpers were delegating to runtime-specific ...WithRuntime variants without adding policy.
- In main.go, bootstrap/runtime struct still contains trivial startServer forwarding methods and function closures.
Next fix: collapse trivial wrapper layers by inlining singleton selection at the public seam and removing duplicate WithRuntime helper entry points where possible.