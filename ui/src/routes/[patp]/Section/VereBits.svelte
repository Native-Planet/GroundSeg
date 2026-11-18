<script>
  // Style
  import "../theme.css"
  import { openModal } from 'svelte-modals'
  import { structure } from '$lib/stores/data'
  import VereBitsModal from '../VereBitsModal.svelte'

  export let patp
  export let ownShip = false

  $: ship = ($structure?.urbits?.[patp]) || {}
  $: vereBits = (ship?.info?.vereBits) || 32
  $: tVereBits = (ship?.transition?.vereBits) || ""
  $: busy = tVereBits.length > 0 && tVereBits != "success" && tVereBits != "error"
  $: statusText = tVereBits == "loading" ? "Preparing..."
    : tVereBits == "stopping" ? "Stopping ship..."
    : tVereBits == "migrating" ? "Migrating snapshot, this can take a while..."
    : tVereBits == "starting" ? "Starting ship..."
    : tVereBits == "success" ? "Switched successfully"
    : tVereBits == "error" ? "Switch failed, the ship stays on its previous runtime. Check migrate.log in the pier."
    : ""

  const select = bits => {
    if (busy || bits == vereBits) {
      return
    }
    openModal(VereBitsModal, {"patp": patp, "currentBits": vereBits, "targetBits": bits, "ownShip": ownShip})
  }
</script>

<div class="section">
  <div class="section-left">
    <div class="section-title">Vere Runtime</div>
    <div class="section-description">
      64-bit Vere removes the 16 GB loom ceiling of the 32-bit runtime. Switching shuts your ship down, migrates its snapshot in place, and starts it again. The ship is offline while this runs.
    </div>
    {#if statusText}
      <div class="status" class:error={tVereBits == "error"} class:ok={tVereBits == "success"}>{statusText}</div>
    {/if}
  </div>
  <div class="section-right">
    <div class="spacer"></div>
    <div class="segmented" class:disabled={busy} role="radiogroup" aria-label="Vere runtime width">
      <button
        type="button"
        role="radio"
        aria-checked={vereBits == 32}
        class:active={vereBits == 32}
        on:click={()=>select(32)}>32-bit</button>
      <button
        type="button"
        role="radio"
        aria-checked={vereBits == 64}
        class:active={vereBits == 64}
        on:click={()=>select(64)}>64-bit</button>
    </div>
  </div>
</div>

<style>
  .section-right {
    display: flex;
    align-items: center;
  }
  .spacer {
    flex: 1;
  }
  .segmented {
    display: inline-flex;
    height: 65px;
    border-radius: 16px;
    background: var(--text-color, #313933);
    padding: 8px;
    gap: 8px;
    user-select: none;
  }
  .segmented button {
    min-width: 96px;
    padding: 0 24px;
    border: 0;
    border-radius: 10px;
    background: transparent;
    color: #FFF;
    font-family: Inter;
    font-size: 24px;
    font-weight: 300;
    letter-spacing: -1.44px;
    cursor: pointer;
    opacity: .7;
  }
  .segmented button.active {
    background: var(--btn-secondary, #FFF);
    color: var(--text-color, #313933);
    opacity: 1;
    cursor: default;
  }
  .segmented.disabled {
    pointer-events: none;
    opacity: .6;
  }
  .status {
    margin: 24px 0 0 8px;
    font-family: Inter;
    font-size: 16px;
    font-weight: 300;
    line-height: 24px;
    letter-spacing: -0.96px;
    color: var(--text-card-color);
  }
  .status.error {
    color: #d45151;
  }
  .status.ok {
    color: #3fa34d;
  }
</style>
