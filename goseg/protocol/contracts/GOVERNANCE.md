# Protocol Contracts Governance

This package is the canonical governance boundary for protocol and StarTram
contract descriptors.

## Scope

- Action and error contract identity and compatibility metadata.
- Family-level catalog composition and validation (`protocol`, `startram`).
- Action-token bindings used by runtime handlers and adapters.

## Authoring Rules

1. Use typed contract APIs (`ContractID`, `ActionNamespace`, `ActionVerb`,
   `ActionContractBinding`) instead of raw string concatenation at call sites.
2. Register contract descriptors through family catalog specs and keep family
   validation active in shared registry assembly.
3. Preserve explicit compatibility metadata for every contract
   (`IntroducedIn`, `DeprecatedIn`, `RemovedIn`, `Compatibility`).
4. Keep public-facing masked error semantics aligned with StarTram contract
   descriptors (`goseg/startram/errors.go`).

## Validation and Tests

Run these checks for contract-surface changes:

```bash
cd goseg
go test ./protocol/contracts ./protocol/actions
```

Runtime contract CI gate coverage is defined in:

- `.github/workflows/upload-contract-gate.yml`
- `.github/scripts/check-upload-contract.sh`
