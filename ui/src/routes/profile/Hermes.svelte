<script>
  import { openModal } from 'svelte-modals'
  import ToggleButton from '$lib/ToggleButton.svelte'
  import { readConfigFile, saveConfigFile } from '$lib/stores/config-files'
  import { hermesInstall, hermesRestart, hermesSave, hermesToggle, hermesUpdate } from '$lib/stores/websocket'
  import { structure } from '$lib/stores/data'
  import WebShellModal from '../[patp]/WebShellModal.svelte'

  const hermesIcon = "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAB4AAAAYCAYAAADtaU2/AAAACXBIWXMAAABtAAAAbgDSdnyfAAAAAXNSR0IB2cksfwAAACBjSFJNAAB6JgAAgIQAAPoAAACA6AAAdTAAAOpgAAA6mAAAF3CculE8AAAABGdBTUEAALGPC/xhBQAABG5JREFUeJyllltIXFcUhkdnvM541zrebx0DGivaor4Eb2gfRBChKAj1wXqp+uKLCo6KgtpRK4goXhJEQRSKVvoiGtNJGySGgjI1GvqgoPRB7MPUCG2Sybjyrx3PMJozdqbZsDn7HM5e315r/Wudo1C4MDw8PBRxcXF3cnJyvo6Pjzd4eXnd9/b2/s7Pz+8rX19frSu2nBowrvb396+LiIjYKioqet3f308mk4nGx8cJQMIrpNFo/goODh7EYdQfDXRzc1MEBASUBAYGvlCpVAIgzYSEBNrd3aWtrS3CwcQzd3d3wgGfKZXKTz4KDCN6T0/Pa0BphoWFUUVFBZ2entLExITtOTynkpKSx9jn/r+gOp2uOTQ0lBA6ioqKopCQEEpJSRH31dXVtLm5SX19fdTV1UUWi4VKS0spNzeXpqenRQp8fHzqXYbitNqVlRXz0tIS1dbWCkPt7e00OjpKAwMDdHBwQDU1NcLD6Ohouri4oIWFBcrMzKS0tDSKiYlhz5/DjtIlMETSmpWVRQUFBaTVagl5Fp4yiEMPFZN9zjs7O6mpqelaKq7ey3UaCmFwbo2KGzll4SQlJcnmm1MCxbOwxIQoxfOgoKAhp8GoVSU2H9gDcRix5iiEh4dfgzJEAl3BbCqHncfsiNMeI7Q/S4YSExNpbGyM6urqqLCwkBoaGkiv19Pg4CBlZ2d/4D1ERUiVWKPG/4QjGqe9xoYvcfJXXDIsruPjY+LBcPZoZGRE3E9NTQkAjFN+fr6obb5nXfAVurBCZOlOg3nA6y9QRpNcQkajUYC4WxkMBtrZ2RH37LXiKtwMlwTH6WAxcprQfL5xCcwDG6taW1sF5PLykm6OxsZGWbFxE2GR8RoOPHAZjHLQz83N0fn5uQBZrVYxpUPk5eXJgtlzeCrWENrvuHeti+Hkkx0dHTQ0NER7e3s2OA+z2SyahxxYcSUyTgH08hZ27joNRZi5g/3EJYQuRkdHRzaPeezv74sy4xqWU7fivaotqO83arX62/8EQrWfY0MUg7HBxAZWV1c/yO/a2powziVWXl4uC8bB/4VAn8HjH7C+HQwP+qHISOQmBNcLNsAda3FxkWZnZ6mnp4fOzs5oeXlZGMdPAVVWVjoC/43SegChmWJjYysRepXD8OKr9AuUuIaXH2FyHVpvGiwuLqbh4WGxrq+vp5aWFlkwxHmI0voVNvYiIyMNOMinDj1OTk6eRHheo/s8R9cyYrNVaoH2kz8KqampDsXFE59SM2BvYeMhPP8Ra51DMPKrw3/VE4R8H2XwEiF6AYW+kYxxU2CV8zg5OaGMjAxbE5Hekflb+Q32uG+73ZpnvKBCuxzFAbZx/eNmuHt7ewV4Y2OD2traqKqqSohsZmZGiG5+fp7KysqE6rmeYecUNvJvhUoDYdEg199D2SPI01N7MPfr5uZm8fFfX18XfySHh4fXVM9/KOnp6Qx+iCh+5hTUfgByDyL7h3svhCdmd3e36GYMZe+3t7dlOxvesyA1yXJ23wFhL8w2DPf/cAAAAABJRU5ErkJggg=="

  $: info = ($structure?.profile?.hermes?.info) || {}
  $: transition = ($structure?.profile?.hermes?.transition) || {}
  $: ships = info?.ships || []
  $: enabled = info?.enabled || false
  $: running = info?.running || false
  $: url = info?.url || "#"
  $: imageInstalled = info?.imageInstalled || false
  $: updateAvailable = info?.updateAvailable || false
  $: versionServerImage = info?.versionServerImage || ""
  $: savedImage = info?.image || ""
  $: selectedImageInstalled = imageInstalled && image.trim() == savedImage
  $: selectedImageChanged = image.trim() != savedImage
  $: providerApiKeySaved = info?.providerApiKeySet || false
  $: savedModelProvider = info?.modelProvider || "openrouter"
  $: providerApiKeyPlaceholder = providerApiKeySaved && modelProvider == savedModelProvider ? "Saved" : ""
  $: providerApiKeyReady = providerApiKey.trim().length > 0 || providerApiKeyPlaceholder.length > 0
  $: webApiKeySaved = info?.webApiKeySet || false
  $: savedWebProvider = info?.webProvider || ""
  $: webProviderUsesUrl = webProvider == "searxng"
  $: webApiKeyPlaceholder = webApiKeySaved && webProvider == savedWebProvider ? "Saved" : ""
  $: webCredentialReady = webProvider.trim().length < 1
    || (webProviderUsesUrl ? webUrl.trim().length > 0 : webApiKey.trim().length > 0 || webApiKeyPlaceholder.length > 0)
  $: apiKeySaved = info?.apiKeySet || false
  $: apiKeyPlaceholder = apiKeySaved ? "Saved" : ""
  $: apiKeyReady = !apiEnabled || apiKey.trim().length > 0 || apiKeySaved
  $: tInstall = transition?.install || ""
  $: tToggle = transition?.toggle || ""
  $: tSave = transition?.save || ""
  $: tRestart = transition?.restart || ""
  $: tError = transition?.error || ""
  $: imageReady = imageInstalled || selectedImageInstalled || tInstall == "success" || tInstall == "installed"
  $: installing = tInstall.length > 0 && tInstall != "success" && tInstall != "error"
  $: selectedShipKey = selectedShip.replace(/^~/, "")
  $: attachedRunning = ($structure?.urbits?.[selectedShipKey]?.info?.running) || false
  $: canConfigure = selectedShip.length > 0 && owner.trim().length > 0 && providerApiKeyReady && webCredentialReady && apiKeyReady
  $: canToggle = enabled || (canConfigure && attachedRunning && imageReady)
  $: busy = tInstall.length > 0 || tToggle.length > 0 || tSave.length > 0 || tRestart.length > 0
  $: dashboardReady = running && url != "#"
  $: activity = tInstall || tToggle || tSave || tRestart
  $: activityText = transitionText(activity)
  $: imageActionMode = selectedImageChanged || !imageInstalled ? "install" : updateAvailable ? "update" : "installed"
  $: installLabel = transitionText(tInstall) || imageActionLabel(imageActionMode)
  $: canInstall = image.trim().length > 0 && !busy && imageActionMode != "installed"

  let selectedShip = ""
  let owner = ""
  let port = 19119
  let image = ""
  let modelProvider = "openrouter"
  let model = "deepseek/deepseek-v4-flash"
  let providerApiKey = ""
  let webProvider = ""
  let webApiKey = ""
  let webUrl = ""
  let apiEnabled = false
  let apiKey = ""
  let dirty = false
  let showAdvanced = false
  let showConfig = false
  let configFile = "hermes/config.yaml"
  let configLoadedFile = ""
  let configLoaded = false
  let configLoading = false
  let configSaving = false
  let configContent = ""
  let configOriginalContent = ""
  let configError = ""
  let configStatus = ""

  const configFiles = [
    { file: "hermes/config.yaml", label: "config.yaml" },
    { file: "hermes/.env", label: ".env" }
  ]

  $: configLabel = configFiles.find(file => file.file == configFile)?.label || "config"
  $: configDirty = configContent !== configOriginalContent
  $: configValidationError = showConfig && !configContent.trim() ? `${configLabel} cannot be empty` : ""
  $: canSaveConfig = configDirty && !configValidationError && !configLoading && !configSaving

  const providers = [
    { value: "openrouter", label: "OpenRouter" },
    { value: "ai-gateway", label: "AI Gateway" },
    { value: "alibaba", label: "Alibaba DashScope" },
    { value: "alibaba-coding-plan", label: "Alibaba Coding Plan" },
    { value: "openai", label: "OpenAI" },
    { value: "anthropic", label: "Anthropic" },
    { value: "arcee", label: "Arcee" },
    { value: "deepseek", label: "DeepSeek" },
    { value: "gmi", label: "GMI Cloud" },
    { value: "huggingface", label: "Hugging Face" },
    { value: "kilocode", label: "Kilo Code" },
    { value: "kimi-coding", label: "Kimi" },
    { value: "kimi-coding-cn", label: "Kimi CN" },
    { value: "nous", label: "Nous" },
    { value: "novita", label: "Novita" },
    { value: "nvidia", label: "NVIDIA NIM" },
    { value: "ollama-cloud", label: "Ollama Cloud" },
    { value: "opencode-go", label: "OpenCode Go" },
    { value: "opencode-zen", label: "OpenCode Zen" },
    { value: "stepfun", label: "StepFun" },
    { value: "xai", label: "xAI" },
    { value: "xiaomi", label: "Xiaomi MiMo" },
    { value: "zai", label: "Z.AI / GLM" }
  ]

  const webProviders = [
    { value: "", label: "Off" },
    { value: "brave-free", label: "Brave Search" },
    { value: "exa", label: "Exa" },
    { value: "firecrawl", label: "Firecrawl" },
    { value: "parallel", label: "Parallel" },
    { value: "searxng", label: "SearXNG" },
    { value: "tavily", label: "Tavily" },
    { value: "xai", label: "xAI" }
  ]

  const transitionLabels = {
    "preparing": "Preparing",
    "removing-container": "Removing container",
    "pulling": "Pulling image",
    "installed": "Installed",
    "loading": "Working",
    "saving": "Saving",
    "validating": "Validating",
    "fetching-code": "Fetching +code",
    "starting": "Starting",
    "stopping": "Stopping",
    "recreating": "Recreating",
    "restarting": "Restarting",
    "success": "Success",
    "error": "Error"
  }

  const transitionText = value => {
    if (!value) return ""
    if (value.startsWith("pulling ")) {
      return `Pulling image ${value.replace("pulling ", "")}`
    }
    return transitionLabels[value] || value
  }

  const imageActionLabel = mode => {
    if (mode == "update") return "Update"
    if (mode == "installed") return "Installed"
    return "Install"
  }

  $: if (tInstall == "success" || tSave == "success" || tToggle == "success") {
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
    webProvider = info?.webProvider || ""
    webApiKey = ""
    webUrl = info?.webUrl || ""
    apiEnabled = info?.apiEnabled || false
    apiKey = ""
  }

  const markDirty = () => {
    dirty = true
  }

  const changeProvider = () => {
    providerApiKey = ""
    markDirty()
  }

  const changeWebProvider = () => {
    webApiKey = ""
    if (webProvider != "searxng") webUrl = ""
    markDirty()
  }

  const toggleAPI = () => {
    apiEnabled = !apiEnabled
    markDirty()
  }

  const payload = () => ({
    ship: selectedShip,
    owner: owner.trim(),
    port: Number(port),
    image: image.trim(),
    modelProvider: modelProvider.trim(),
    model: model.trim(),
    providerApiKey: providerApiKey.trim(),
    webProvider: webProvider.trim(),
    webApiKey: webApiKey.trim(),
    webUrl: webUrl.trim(),
    apiEnabled,
    apiKey: apiKey.trim()
  })

  const save = () => {
    if (busy) return
    hermesSave(payload())
  }

  const install = () => {
    if (!canInstall) return
    if (imageActionMode == "update") {
      hermesUpdate(payload())
    } else {
      hermesInstall(payload())
    }
  }

  const toggle = () => {
    if (!canToggle || tToggle.length > 0) return
    if (!enabled && busy) return
    hermesToggle(payload())
  }

  const toggleConfig = async () => {
    showConfig = !showConfig
    if (showConfig && (!configLoaded || configLoadedFile != configFile)) {
      await loadHermesConfig()
    }
  }

  const changeConfigFile = async () => {
    configLoaded = false
    configLoadedFile = ""
    configContent = ""
    configOriginalContent = ""
    configStatus = ""
    configError = ""
    if (showConfig) {
      await loadHermesConfig()
    }
  }

  const loadHermesConfig = async () => {
    configLoading = true
    configError = ""
    configStatus = ""
    try {
      const response = await readConfigFile(configFile)
      configContent = response.content || ""
      configOriginalContent = configContent
      configLoaded = true
      configLoadedFile = configFile
    } catch (err) {
      configError = err.message
    } finally {
      configLoading = false
    }
  }

  const saveHermesConfig = async () => {
    if (!canSaveConfig) return
    configSaving = true
    configError = ""
    configStatus = ""
    try {
      const response = await saveConfigFile(configFile, configContent)
      configContent = response.content || configContent
      configOriginalContent = configContent
      configStatus = `${configLabel} saved`
    } catch (err) {
      configError = err.message
    } finally {
      configSaving = false
    }
  }

  const openHermesTerminal = () => {
    openModal(WebShellModal, {
      target: "hermes",
      title: "Hermes Terminal",
      width: 1180,
      height: "72vh",
      minHeight: 560,
    })
  }
</script>

<div class="container">
  <div class="top">
    <div>
      <div class="title-row">
        <img src={hermesIcon} alt="" aria-hidden="true" />
        <div class="prof-title">HERMES</div>
      </div>
      <div class="status-row">
        <div class="status" class:on={enabled}>{enabled ? "Enabled" : "Disabled"}</div>
        <div class="status" class:on={running}>{running ? "Running" : "Stopped"}</div>
        <div class="status" class:on={imageReady} class:warn={updateAvailable && !selectedImageChanged}>
          {#if updateAvailable && !selectedImageChanged}
            Update available
          {:else}
            {imageReady ? "Installed" : "Not installed"}
          {/if}
        </div>
      </div>
    </div>
    <div class="spacer"></div>
    <div class="controls">
      <button
        class="install"
        disabled={!canInstall}
        class:success={imageActionMode == "installed"}
        on:click={install}>
        {installLabel}
      </button>
      <ToggleButton on:click={toggle} loading={tToggle.length > 0} disabled={!canToggle} on={enabled} />
      <a
        class="dashboard"
        class:hidden={!dashboardReady}
        href={dashboardReady ? url : "#"}
        target="_blank"
        tabindex={dashboardReady ? 0 : -1}
        aria-disabled={!dashboardReady}>
        Open
      </a>
      <button
        class="restart"
        disabled={!enabled || !providerApiKeySaved || !imageReady || tRestart.length > 0}
        class:success={tRestart == "success"}
        on:click={hermesRestart}>
        {#if tRestart.length > 0}
          {transitionText(tRestart)}
        {:else}
          Restart
        {/if}
      </button>
    </div>
  </div>

  {#if tError.length > 0 || activityText.length > 0}
    <div class="message" class:error={tError.length > 0 || activity == "error"}>
      {tError || activityText}
    </div>
  {/if}

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

  <div class="grid">
    <label>
      <span>Web Search</span>
      <select bind:value={webProvider} on:change={changeWebProvider}>
        {#each webProviders as provider}
          <option value={provider.value}>{provider.label}</option>
        {/each}
      </select>
    </label>
    {#if webProviderUsesUrl}
      <label>
        <span>SearXNG URL</span>
        <input
          bind:value={webUrl}
          on:input={markDirty}
          placeholder="http://localhost:8888" />
      </label>
    {:else}
      <label>
        <span>Web API Key</span>
        <input
          type="password"
          autocomplete="off"
          bind:value={webApiKey}
          on:input={markDirty}
          disabled={webProvider.trim().length < 1}
          placeholder={webProvider.trim().length < 1 ? "" : webApiKeyPlaceholder} />
      </label>
    {/if}
  </div>

  <button class="advanced-toggle" class:active={showAdvanced} on:click={()=>showAdvanced = !showAdvanced}>
    Advanced
  </button>

  {#if showAdvanced}
    <div class="advanced">
      <label>
        <span>Image</span>
        <input bind:value={image} on:input={markDirty} disabled={installing} />
      </label>
      <label>
        <span>Port</span>
        <input type="number" min="1" max="65535" bind:value={port} on:input={markDirty} />
      </label>
      <label>
        <span>API Server Key</span>
        <input
          type="password"
          autocomplete="off"
          bind:value={apiKey}
          on:input={markDirty}
          disabled={!apiEnabled}
          placeholder={apiEnabled ? apiKeyPlaceholder : ""} />
      </label>
      <div class="api-toggle-row">
        <span>API Server</span>
        <ToggleButton on:click={toggleAPI} on={apiEnabled} />
      </div>
      <div class="versions">
        <div>Hermes {info?.hermesVersion || ""}</div>
        <div>Tlon Adapter {info?.tlonAdapterVersion || ""}</div>
        {#if versionServerImage}
          <div>Latest {versionServerImage}</div>
        {/if}
      </div>
    </div>
  {/if}

  <button class="config-toggle" class:active={showConfig} on:click={toggleConfig}>
    <span>Runtime files</span>
    <span>{showConfig ? "Hide" : "Edit"}</span>
  </button>

  {#if showConfig}
    <div class="config-editor">
      <div class="config-toolbar">
        <select
          class="config-select"
          bind:value={configFile}
          on:change={changeConfigFile}
          disabled={configLoading || configSaving}>
          {#each configFiles as file}
            <option value={file.file}>{file.label}</option>
          {/each}
        </select>
        <button on:click={openHermesTerminal} disabled={!running}>Terminal</button>
        <button on:click={loadHermesConfig} disabled={configLoading || configSaving}>Reload</button>
        <button class="save-config" on:click={saveHermesConfig} disabled={!canSaveConfig}>
          {configSaving ? "Saving" : "Save"}
        </button>
      </div>

      {#if configLoading}
        <div class="message">Loading {configLabel}...</div>
      {/if}
      {#if configError}
        <div class="message error">{configError}</div>
      {:else if configValidationError}
        <div class="message error">{configValidationError}</div>
      {:else if configStatus}
        <div class="message success">{configStatus}</div>
      {:else if configDirty}
        <div class="message">Unsaved {configLabel} edits</div>
      {/if}

      <textarea
        class="config-textarea"
        spellcheck="false"
        bind:value={configContent}
        disabled={configLoading || configSaving}
        aria-label={`Hermes ${configLabel} editor`}
      />
    </div>
  {/if}

  <div class="actions">
    <button
      class="save"
      disabled={!dirty || !canConfigure || busy}
      class:success={tSave == "success"}
      on:click={save}>
      {#if tSave.length > 0}
        {transitionText(tSave)}
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
    display: grid;
    grid-template-columns: 1fr;
    align-items: flex-start;
    gap: 20px;
  }
  .title-row {
    display: flex;
    align-items: center;
    gap: 8px;
  }
  .title-row img {
    width: 30px;
    height: 30px;
    object-fit: contain;
    flex: 0 0 30px;
  }
  .status-row {
    display: flex;
    gap: 12px;
    margin-top: 8px;
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
  .status.warn {
    background: #D9A100;
    color: #fff;
  }
  .controls {
    display: grid;
    grid-template-columns: 168px 135px 96px 150px;
    grid-auto-rows: 65px;
    align-items: center;
    gap: 12px;
    justify-content: stretch;
    height: 65px;
    width: 585px;
    max-width: 100%;
    overflow: hidden;
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
  .config-toggle {
    width: 100%;
    margin-top: 24px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    border-radius: 12px;
    background: transparent;
    color: var(--NP_Black, #313933);
    font-family: Inter;
    font-size: 18px;
    font-weight: 300;
    letter-spacing: 0;
    padding: 0;
  }
  .config-toggle span:first-child {
    color: var(--Gray-400, #5C7060);
    font-size: 18px;
  }
  .config-toggle span:last-child {
    background: var(--btn-secondary);
    border-radius: 12px;
    color: #fff;
    padding: 12px 24px;
  }
  .config-toggle.active span:last-child {
    background: black;
  }
  .config-editor {
    margin-top: 16px;
  }
  .config-toolbar {
    display: grid;
    grid-template-columns: minmax(160px, 220px) 120px 96px 96px;
    justify-content: end;
    gap: 12px;
  }
  .config-select {
    height: 48px;
    border-radius: 8px;
    font-size: 16px;
    padding: 0 12px;
  }
  .config-toolbar button {
    height: 48px;
    border-radius: 8px;
    background: var(--Gray-400, #5C7060);
    color: #fff;
    font-family: Inter;
    font-size: 16px;
    font-weight: 300;
    padding: 0 20px;
    white-space: nowrap;
  }
  .config-toolbar button.save-config {
    background: #077D13;
  }
  .config-textarea {
    width: 100%;
    min-height: 360px;
    box-sizing: border-box;
    margin-top: 16px;
    border: 1px solid var(--Gray-200, #ABBAAE);
    border-radius: 8px;
    background: #101511;
    color: #F8F8F6;
    font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", monospace;
    font-size: 14px;
    line-height: 20px;
    letter-spacing: 0;
    padding: 16px;
    resize: vertical;
  }
  .advanced {
    display: grid;
    grid-template-columns: 1fr 180px;
    gap: 24px;
    margin-top: 24px;
  }
  .api-toggle-row {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }
  .versions {
    grid-column: 1 / -1;
    display: flex;
    flex-wrap: wrap;
    gap: 24px;
    color: var(--Gray-400, #5C7060);
    font-family: Inter;
    font-size: 16px;
    overflow-wrap: anywhere;
  }
  .actions {
    display: flex;
    justify-content: flex-end;
    margin-top: 32px;
    min-height: 65px;
  }
  .save, .restart, .install, .dashboard {
    border-radius: 16px;
    background: black;
    color: #fff;
    font-family: Inter;
    font-size: 20px;
    font-weight: 300;
    letter-spacing: 0;
    height: 65px;
    padding: 0 12px;
    cursor: pointer;
    overflow: hidden;
    text-align: center;
    text-overflow: ellipsis;
    white-space: nowrap;
    box-sizing: border-box;
    width: 100%;
  }
  .save {
    width: 180px;
    padding: 0;
    font-size: 24px;
  }
  .dashboard {
    display: flex;
    align-items: center;
    justify-content: center;
    text-decoration: none;
  }
  .dashboard.hidden {
    visibility: hidden;
    pointer-events: none;
  }
  .save:disabled, .restart:disabled, .install:disabled,
  .config-toolbar button:disabled,
  .config-textarea:disabled {
    opacity: .5;
    pointer-events: none;
  }
  .message {
    margin-top: 24px;
    border-radius: 12px;
    background: var(--Gray-100, #DDE3DF);
    color: var(--text-color);
    font-family: Inter;
    font-size: 16px;
    padding: 14px 18px;
  }
  .message.error {
    background: #F9D2D2;
    color: #8A1111;
  }
  .message.success {
    color: #077D13;
    opacity: 1;
    pointer-events: auto;
  }
  .success {
    opacity: .7;
    pointer-events: none;
  }
  .spacer {
    display: none;
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
    .controls,
    .actions {
      justify-content: flex-start;
      width: 100%;
    }
    .controls {
      grid-template-columns: minmax(0, 1fr) 135px;
      grid-auto-rows: 65px;
      height: 142px;
      overflow: visible;
    }
    .config-toolbar {
      grid-template-columns: 1fr 1fr;
      justify-content: stretch;
    }
    .save, .restart, .install, .dashboard {
      font-size: 20px;
    }
  }
</style>
