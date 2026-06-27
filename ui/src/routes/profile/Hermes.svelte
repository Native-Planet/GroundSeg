<script>
  import ToggleButton from '$lib/ToggleButton.svelte'
  import { hermesRestart, hermesSave, hermesToggle } from '$lib/stores/websocket'
  import { structure } from '$lib/stores/data'

  $: info = ($structure?.profile?.hermes?.info) || {}
  $: transition = ($structure?.profile?.hermes?.transition) || {}
  $: ships = info?.ships || []
  $: enabled = info?.enabled || false
  $: running = info?.running || false
  $: url = info?.url || "#"
  $: providerApiKeySaved = info?.providerApiKeySet || false
  $: savedModelProvider = info?.modelProvider || "openrouter"
  $: providerApiKeyPlaceholder = providerApiKeySaved && modelProvider == savedModelProvider ? "Saved" : ""
  $: providerApiKeyReady = providerApiKey.trim().length > 0 || providerApiKeyPlaceholder.length > 0
  $: tToggle = transition?.toggle || ""
  $: tSave = transition?.save || ""
  $: tRestart = transition?.restart || ""
  $: selectedShipKey = selectedShip.replace(/^~/, "")
  $: attachedRunning = ($structure?.urbits?.[selectedShipKey]?.info?.running) || false
  $: canConfigure = selectedShip.length > 0 && owner.trim().length > 0 && providerApiKeyReady
  $: canToggle = enabled || (canConfigure && attachedRunning)
  $: busy = tToggle.length > 0 || tSave.length > 0 || tRestart.length > 0
  $: dashboardReady = running && url != "#"

  let selectedShip = ""
  let owner = ""
  let port = 19119
  let image = ""
  let modelProvider = "openrouter"
  let model = "deepseek/deepseek-v4-flash"
  let providerApiKey = ""
  let dirty = false
  let showAdvanced = false

  const providers = [
    { value: "openrouter", label: "OpenRouter" },
    { value: "openai", label: "OpenAI" },
    { value: "anthropic", label: "Anthropic" }
  ]

  $: if (tSave == "success" || tToggle == "success") {
    dirty = false
  }

  $: if (!dirty) {
    selectedShip = info?.ship || ships[0] || ""
    owner = info?.owner || ""
    port = info?.port || 19119
    image = info?.image || ""
    modelProvider = info?.modelProvider || "openrouter"
    model = info?.model || "deepseek/deepseek-v4-flash"
    providerApiKey = ""
  }

  const markDirty = () => {
    dirty = true
  }

  const changeProvider = () => {
    providerApiKey = ""
    markDirty()
  }

  const payload = () => ({
    ship: selectedShip,
    owner: owner.trim(),
    port: Number(port),
    image: image.trim(),
    modelProvider: modelProvider.trim(),
    model: model.trim(),
    providerApiKey: providerApiKey.trim()
  })

  const save = () => {
    hermesSave(payload())
  }

  const toggle = () => {
    if (!canToggle || busy) return
    hermesToggle(payload())
  }
</script>

<div class="container">
  <div class="top">
    <div>
      <div class="prof-title">HERMES</div>
      <div class="status-row">
        <div class="status" class:on={enabled}>{enabled ? "Enabled" : "Disabled"}</div>
        <div class="status" class:on={running}>{running ? "Running" : "Stopped"}</div>
      </div>
    </div>
    <div class="spacer"></div>
    <div class="controls">
      <ToggleButton on:click={toggle} loading={busy} disabled={!canToggle} on={enabled} />
      {#if dashboardReady}
        <a class="dashboard" href={url} target="_blank">Open</a>
      {/if}
      <button
        class="restart"
        disabled={!enabled || !providerApiKeySaved || tRestart.length > 0}
        class:success={tRestart == "success"}
        on:click={hermesRestart}>
        {#if tRestart == "loading"}
          Restarting
        {:else if tRestart == "success"}
          Success
        {:else if tRestart == "error"}
          Error
        {:else}
          Restart
        {/if}
      </button>
    </div>
  </div>

  <div class="grid">
    <label>
      <span>Ship</span>
      <select bind:value={selectedShip} on:change={markDirty}>
        <option value="">Select ship</option>
        {#each ships as ship}
          <option value={ship}>{ship}</option>
        {/each}
      </select>
    </label>
    <label>
      <span>Owner</span>
      <input bind:value={owner} on:input={markDirty} placeholder="~sampel-palnet" />
    </label>
  </div>

  <div class="grid">
    <label>
      <span>Provider</span>
      <select bind:value={modelProvider} on:change={changeProvider}>
        {#each providers as provider}
          <option value={provider.value}>{provider.label}</option>
        {/each}
      </select>
    </label>
    <label>
      <span>Model</span>
      <input bind:value={model} on:input={markDirty} />
    </label>
  </div>

  <div class="grid key-grid">
    <label>
      <span>API Key</span>
      <input
        type="password"
        autocomplete="off"
        bind:value={providerApiKey}
        on:input={markDirty}
        placeholder={providerApiKeyPlaceholder} />
    </label>
  </div>

  <button class="advanced-toggle" class:active={showAdvanced} on:click={()=>showAdvanced = !showAdvanced}>
    Advanced
  </button>

  {#if showAdvanced}
    <div class="advanced">
      <label>
        <span>Image</span>
        <input bind:value={image} on:input={markDirty} />
      </label>
      <label>
        <span>Port</span>
        <input type="number" min="1" max="65535" bind:value={port} on:input={markDirty} />
      </label>
      <div class="versions">
        <div>Hermes {info?.hermesVersion || ""}</div>
        <div>Tlon Adapter {info?.tlonAdapterVersion || ""}</div>
      </div>
    </div>
  {/if}

  <div class="actions">
    <button
      class="save"
      disabled={!dirty || !canConfigure || tSave.length > 0}
      class:success={tSave == "success"}
      on:click={save}>
      {#if tSave == "loading"}
        Saving
      {:else if tSave == "success"}
        Saved
      {:else if tSave == "error"}
        Error
      {:else}
        Save
      {/if}
    </button>
  </div>
</div>

<style>
  .container {
    margin: auto;
    width: calc(1104px - (56px * 2));
    max-width: 98vw;
    padding: 56px;
  }
  .top {
    display: flex;
    gap: 24px;
    flex-wrap: wrap;
  }
  .status-row {
    display: flex;
    gap: 12px;
  }
  .status {
    background: var(--Gray-100, #DDE3DF);
    border-radius: 10px;
    color: var(--text-color);
    font-family: Inter;
    font-size: 16px;
    padding: 10px 14px;
  }
  .status.on {
    background: #077D13;
    color: #fff;
  }
  .controls {
    display: flex;
    align-items: center;
    gap: 16px;
    flex-wrap: wrap;
    justify-content: flex-end;
  }
  .grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 24px;
    margin-top: 32px;
  }
  .key-grid {
    grid-template-columns: 1fr;
  }
  label {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }
  span {
    color: var(--Gray-400, #5C7060);
    font-family: Inter;
    font-size: 18px;
    font-weight: 300;
  }
  input, select {
    height: 65px;
    border: 0;
    border-radius: 16px;
    background: var(--bg-modal);
    color: var(--NP_Black, #313933);
    font-family: Inter;
    font-size: 24px;
    font-weight: 300;
    letter-spacing: 0;
    padding: 0 24px;
  }
  input:focus, select:focus {
    outline: 2px solid var(--btn-secondary);
  }
  .advanced-toggle {
    margin-top: 32px;
    background: var(--btn-secondary);
    border-radius: 12px;
    color: #fff;
    font-family: Inter;
    font-size: 18px;
    padding: 12px 24px;
  }
  .advanced-toggle.active {
    background: black;
  }
  .advanced {
    display: grid;
    grid-template-columns: 1fr 180px;
    gap: 24px;
    margin-top: 24px;
  }
  .versions {
    grid-column: 1 / -1;
    display: flex;
    gap: 24px;
    color: var(--Gray-400, #5C7060);
    font-family: Inter;
    font-size: 16px;
  }
  .actions {
    display: flex;
    justify-content: flex-end;
    margin-top: 32px;
  }
  .save, .restart, .dashboard {
    border-radius: 16px;
    background: black;
    color: #fff;
    font-family: Inter;
    font-size: 24px;
    font-weight: 300;
    letter-spacing: 0;
    height: 65px;
    padding: 0 48px;
    cursor: pointer;
  }
  .dashboard {
    display: flex;
    align-items: center;
    text-decoration: none;
  }
  .save:disabled, .restart:disabled {
    opacity: .5;
    pointer-events: none;
  }
  .success {
    opacity: .7;
    pointer-events: none;
  }
  .spacer {
    flex: 1;
  }

  @media (max-width: 760px) {
    .container {
      padding: 32px 16px;
      width: 100%;
    }
    .grid,
    .advanced {
      grid-template-columns: 1fr;
    }
    .top {
      align-items: flex-start;
    }
    .spacer {
      display: none;
    }
    .controls,
    .actions {
      justify-content: flex-start;
      width: 100%;
    }
    .save, .restart, .dashboard {
      font-size: 20px;
      padding: 0 24px;
    }
  }
</style>
