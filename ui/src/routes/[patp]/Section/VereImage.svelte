<script>
  import "../theme.css"
  import { structure, URBIT_MODE } from '$lib/stores/data'
  import { setVereTag } from '$lib/stores/websocket'

  export let patp
  export let ownShip = false
  export let urbitVersion = ""
  export let urbitImageTagOverride = ""
  export let versionServerVereTag = ""
  export let vereTags = []

  let expanded = false
  let draft = urbitImageTagOverride
  let lastSynced = urbitImageTagOverride

  $: if (urbitImageTagOverride !== lastSynced) {
    draft = urbitImageTagOverride
    lastSynced = urbitImageTagOverride
  }

  $: tVereTag = ($structure?.urbits?.[patp]?.transition?.vereTag) || ""
  $: isLoading = tVereTag == "loading"
  $: isSuccess = tVereTag == "success"
  $: remoteMessage = tVereTag.length > 0 && !isLoading && !isSuccess ? tVereTag : ""
  $: tags = [...new Set([versionServerVereTag, urbitVersion, urbitImageTagOverride, ...vereTags].filter(Boolean))]
  $: activeLabel = urbitImageTagOverride.length > 0 ? urbitImageTagOverride : `Version server (${versionServerVereTag || urbitVersion || "unknown"})`
  $: saveDisabled = isLoading || isSuccess || draft == urbitImageTagOverride
  $: toggleLabel = expanded ? "Collapse" : "Change"

  const handleSave = () => {
    setVereTag(patp, draft)
  }
</script>

<div class="wrapper">
  <div class="section">
    <div class="section-left">
      <div class="section-title">Vere Image</div>
      <div class="section-description">{activeLabel}</div>
    </div>
    <div class="section-right">
      <button class="toggle-button" on:click={() => expanded = !expanded}>
        {toggleLabel}
      </button>
    </div>
  </div>

  {#if expanded}
    <div class="panel">
      {#if $URBIT_MODE && ownShip}
        <div class="helper warning">
          Saving rebuilds this ship and may temporarily disconnect the current GroundSeg session.
        </div>
      {/if}
      <select bind:value={draft} disabled={isLoading}>
        <option value="">Version server ({versionServerVereTag || "current"})</option>
        {#each tags as tag}
          <option value={tag}>{tag}</option>
        {/each}
      </select>
      {#if remoteMessage.length > 0}
        <div class="status-message">{remoteMessage}</div>
      {/if}
      <div class="actions">
        <button class="save-button" disabled={saveDisabled} on:click={handleSave}>
          {#if isLoading}
            Saving...
          {:else if isSuccess}
            Saved!
          {:else}
            Save
          {/if}
        </button>
      </div>
    </div>
  {/if}
</div>

<style>
  .wrapper {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }
  .panel {
    margin-left: auto;
    width: min(100%, 760px);
    display: flex;
    flex-direction: column;
    gap: 12px;
  }
  .helper {
    color: var(--text-card-color);
    font-size: 14px;
    line-height: 20px;
  }
  .warning {
    color: #f1b96b;
  }
  select {
    height: 65px;
    border-radius: 16px;
    color: var(--Gray-200, #ABBAAE);
    font-family: Inter;
    font-size: 20px;
    font-weight: 300;
    padding: 0 20px;
    border: 2px solid var(--text-card-color);
    background: var(--bg-card);
  }
  select:disabled {
    opacity: .6;
    pointer-events: none;
  }
  .actions {
    display: flex;
    justify-content: flex-end;
  }
  .toggle-button,
  .save-button {
    border-radius: 16px;
    cursor: pointer;
    font-family: Inter;
    font-size: 20px;
    font-weight: 300;
    letter-spacing: 0;
  }
  .toggle-button {
    padding: 18px 28px;
    color: var(--text-card-color);
    background: #2C3A2E;
  }
  .save-button {
    padding: 18px 32px;
    color: var(--Gray-200, #ABBAAE);
    background: #2C3A2E;
  }
  .toggle-button:disabled,
  .save-button:disabled {
    opacity: .6;
    pointer-events: none;
  }
  .status-message {
    color: #d45151;
    font-size: 14px;
    line-height: 20px;
  }
</style>
