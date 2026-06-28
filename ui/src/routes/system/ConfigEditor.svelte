<script>
  import { URBIT_MODE } from '$lib/stores/data'
  import { listConfigFiles, readConfigFile, saveConfigFile } from '$lib/stores/config-files'

  let open = false
  let loaded = false
  let loading = false
  let saving = false
  let files = []
  let selected = ''
  let content = ''
  let originalContent = ''
  let error = ''
  let status = ''

  $: dirty = content !== originalContent
  $: validationError = validateJSON(content)
  $: canSave = Boolean(selected && dirty && !validationError && !loading && !saving)

  function validateJSON(value) {
    if (!open || !selected) return ''
    if (!value.trim()) return 'JSON cannot be empty'
    try {
      JSON.parse(value)
      return ''
    } catch (err) {
      return err.message
    }
  }

  async function toggleOpen() {
    open = !open
    if (open && !loaded && !$URBIT_MODE) {
      await loadFiles()
    }
  }

  async function loadFiles() {
    loading = true
    error = ''
    status = ''
    try {
      const response = await listConfigFiles()
      files = response.files || []
      selected = files[0]?.file || ''
      loaded = true
      if (selected) {
        await loadFile(selected)
      }
    } catch (err) {
      error = err.message
    } finally {
      loading = false
    }
  }

  async function loadFile(file) {
    if (!file) return
    loading = true
    error = ''
    status = ''
    try {
      const response = await readConfigFile(file)
      content = response.content || ''
      originalContent = content
    } catch (err) {
      error = err.message
    } finally {
      loading = false
    }
  }

  async function handleFileChange(event) {
    const next = event.currentTarget.value
    if (dirty && !window.confirm('Discard unsaved config edits?')) {
      event.currentTarget.value = selected
      return
    }
    selected = next
    await loadFile(selected)
  }

  function formatJSON() {
    error = ''
    status = ''
    try {
      content = JSON.stringify(JSON.parse(content), null, 4)
    } catch (err) {
      error = err.message
    }
  }

  async function saveFile() {
    if (!canSave) return
    saving = true
    error = ''
    status = ''
    try {
      const response = await saveConfigFile(selected, content)
      content = response.content || content
      originalContent = content
      status = `${selected} saved`
    } catch (err) {
      error = err.message
    } finally {
      saving = false
    }
  }
</script>

<div class="container">
  <button class="header" on:click={toggleOpen} aria-expanded={open}>
    <span>ADVANCED CONFIG</span>
    <span>{open ? 'Hide' : 'Edit JSON'}</span>
  </button>

  {#if open}
    {#if $URBIT_MODE}
      <div class="message error">Config editor is only available from GroundSeg.</div>
    {:else}
      <div class="toolbar">
        <select bind:value={selected} on:change={handleFileChange} disabled={loading || saving || files.length === 0}>
          {#each files as file}
            <option value={file.file}>{file.label} ({file.file})</option>
          {/each}
        </select>
        <button on:click={loadFiles} disabled={loading || saving}>Reload</button>
        <button on:click={formatJSON} disabled={loading || saving || Boolean(validationError)}>Format</button>
        <button class="save" on:click={saveFile} disabled={!canSave}>{saving ? 'Saving...' : 'Save'}</button>
      </div>

      {#if loading}
        <div class="message">Loading config...</div>
      {/if}
      {#if error}
        <div class="message error">{error}</div>
      {:else if validationError}
        <div class="message error">{validationError}</div>
      {:else if status}
        <div class="message success">{status}</div>
      {:else if dirty}
        <div class="message">Unsaved edits</div>
      {/if}

      <textarea
        spellcheck="false"
        bind:value={content}
        disabled={loading || saving || !selected}
        aria-label="Config JSON editor"
      />
    {/if}
  {/if}
</div>

<style>
  .container {
    margin: 0;
  }
  .header {
    width: 100%;
    display: flex;
    align-items: center;
    justify-content: space-between;
    border-radius: 16px;
    cursor: pointer;
    background: transparent;
    color: var(--NP_Black, #161D17);
    font-family: BPdotsUnicase;
    font-size: 32px;
    font-style: normal;
    font-weight: 700;
    letter-spacing: 0;
    padding: 0;
  }
  .header span:last-child {
    color: #FFF;
    background: var(--btn-secondary);
    border-radius: 16px;
    font-family: Inter;
    font-size: 20px;
    font-weight: 300;
    line-height: 32px;
    padding: 12px 28px;
  }
  .toolbar {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-top: 32px;
  }
  select {
    flex: 1;
    min-width: 0;
    height: 48px;
    border: 1px solid var(--Gray-200, #ABBAAE);
    border-radius: 8px;
    background: #FFF;
    color: var(--NP_Black, #161D17);
    font-family: Inter;
    font-size: 16px;
    letter-spacing: 0;
    padding: 0 12px;
  }
  button {
    border: 0;
  }
  .toolbar button {
    height: 48px;
    border-radius: 8px;
    cursor: pointer;
    background: var(--Gray-400, #5C7060);
    color: #FFF;
    font-family: Inter;
    font-size: 16px;
    font-weight: 300;
    letter-spacing: 0;
    padding: 0 20px;
  }
  .toolbar button.save {
    background: #077D13;
  }
  button:disabled,
  select:disabled,
  textarea:disabled {
    opacity: 0.6;
    pointer-events: none;
  }
  .message {
    margin-top: 16px;
    color: var(--NP_Black, #161D17);
    font-family: Inter;
    font-size: 16px;
    font-weight: 300;
    letter-spacing: 0;
  }
  .error {
    color: #B00020;
  }
  .success {
    color: #077D13;
  }
  textarea {
    width: 100%;
    min-height: 420px;
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
</style>
