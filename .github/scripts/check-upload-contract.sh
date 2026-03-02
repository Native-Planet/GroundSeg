#!/usr/bin/env bash
set -euo pipefail

determine_base_ref() {
  if [[ -n "${GITHUB_BASE_REF:-}" ]]; then
    git fetch --no-tags --depth=1 origin "${GITHUB_BASE_REF}"
    echo "origin/${GITHUB_BASE_REF}"
    return
  fi

  if git rev-parse --verify HEAD~1 >/dev/null 2>&1; then
    echo "HEAD~1"
    return
  fi

  echo ""
}

changed() {
  local path="$1"
  grep -qx "${path}" <<<"${changed_files}"
}

base_ref="$(determine_base_ref)"
if [[ -z "${base_ref}" ]]; then
  echo "No base ref available (first commit). Skipping runtime contract gate."
  exit 0
fi

changed_files="$(git diff --name-only "${base_ref}"...HEAD)"

upload_handler_changed=0
upload_service_changed=0
startram_errors_changed=0
protocol_actions_changed=0
if changed "goseg/handler/ws/upload.go"; then
  upload_handler_changed=1
fi
if changed "goseg/uploadsvc/service.go"; then
  upload_service_changed=1
fi
if changed "goseg/startram/errors.go"; then
  startram_errors_changed=1
fi
if changed "goseg/protocol/actions/actions.go"; then
  protocol_actions_changed=1
fi

if [[ ${upload_handler_changed} -eq 0 && ${upload_service_changed} -eq 0 && ${startram_errors_changed} -eq 0 && ${protocol_actions_changed} -eq 0 ]]; then
  echo "Runtime contract surfaces unchanged; contract gate passed."
  exit 0
fi

if [[ ${upload_handler_changed} -eq 1 ]] && ! changed "goseg/handler/ws/upload_test.go"; then
  echo "Upload handler contract violation:"
  echo "  goseg/handler/ws/upload.go changed without goseg/handler/ws/upload_test.go update."
  echo "Required matrix: decode failure, open-endpoint success/failure, reset success/failure, unsupported action."
  exit 1
fi

if [[ ${upload_service_changed} -eq 1 ]] && ! changed "goseg/uploadsvc/service_test.go"; then
  echo "Upload service contract violation:"
  echo "  goseg/uploadsvc/service.go changed without goseg/uploadsvc/service_test.go update."
  echo "Required coverage: dispatch-table parity across SupportedActions and unsupported-action rejection."
  exit 1
fi

if [[ ${startram_errors_changed} -eq 1 ]] && ! changed "goseg/startram/errors_test.go"; then
  echo "Startram masked-error contract violation:"
  echo "  goseg/startram/errors.go changed without goseg/startram/errors_test.go update."
  echo "Required coverage: redacted outward message plus errors.Is and errors.Unwrap assertions."
  exit 1
fi

if [[ ${protocol_actions_changed} -eq 1 ]] && ! changed "goseg/protocol/actions/actions_test.go"; then
  echo "Protocol actions contract violation:"
  echo "  goseg/protocol/actions/actions.go changed without goseg/protocol/actions/actions_test.go update."
  echo "Required coverage: namespace contract parity and unsupported-action behavior."
  exit 1
fi

echo "Runtime contract files changed with matching tests. Running targeted contract suites."
cd goseg

go test ./handler/ws -run 'TestUploadHandlerBranchMatrix|TestUploadHandlerDispatchesActions|TestUploadHandlerPropagatesServiceErrors|TestUploadHandlerRejectsUnknownAction|TestNewUploadMessageHandlerNilServiceAndUploadRejectsMalformedJSON' -count=1
go test ./uploadsvc -run 'TestExecutorDispatchTableParityAcrossSupportedActions|TestExecutorSupportedActionsMatchesContract|TestExecutorReturnsUnsupportedActionError' -count=1
go test ./startram -run 'TestWrapAPIConnectionErrorRedactsUpstreamDetailsAndPreservesCause|TestWrapAPIConnectionErrorRetainsStableMessageWithoutPubkey' -count=1
go test ./protocol/actions -run 'TestParseUploadActionRejectsUnknown|TestSupportedUploadActionsMatchesContract|TestSupportedC2CActionsMatchesContract' -count=1
