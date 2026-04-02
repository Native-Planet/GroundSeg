<script>
  import "../theme.css"
  import { structure, URBIT_MODE } from '$lib/stores/data'
  import { setUrbitExtraArgs } from '$lib/stores/websocket'
  import { lintPersistentArgs } from '$lib/urbitArgs'

  export let patp
  export let ownShip = false
  export let extraArgs = ""
  export let bootCommandBase = ""

  let expanded = false
  let draft = extraArgs
  let lastSynced = extraArgs

  $: if (extraArgs !== lastSynced) {
    draft = extraArgs
    lastSynced = extraArgs
  }

  $: trimmedDraft = draft.trim()
  $: hasSavedArgs = extraArgs.trim().length > 0
  $: lint = lintPersistentArgs(draft)
  $: tExtraArgs = ($structure?.urbits?.[patp]?.transition?.extraArgs) || ""
  $: isLoading = tExtraArgs == "loading"
  $: isSuccess = tExtraArgs == "success"
  $: remoteMessage = tExtraArgs.length > 0 && !isLoading && !isSuccess ? tExtraArgs : ""
  $: statusMessage = lint.message.length > 0 ? lint.message : remoteMessage
  $: preview = bootCommandBase.length > 0
    ? (trimmedDraft.length > 0 ? `${bootCommandBase} ${trimmedDraft}` : bootCommandBase)
    : "Preview unavailable"
  $: saveDisabled = isLoading || isSuccess || !lint.valid || trimmedDraft == extraArgs
  $: toggleLabel = expanded ? "Collapse" : (hasSavedArgs ? "Edit" : "Add Flags")

  const handleSave = () => {
    setUrbitExtraArgs(patp, trimmedDraft)
  }
</script>

<div class="wrapper">
  <div class="section">
    <div class="section-left">
      <div class="section-title">Additional CLI Flags</div>
      <div class="section-description">
        Optional flags appended to GroundSeg's generated <code>urbit</code> command.
      </div>
    </div>
    <div class="section-right">
      <button class="toggle-button" on:click={() => expanded = !expanded}>
        {toggleLabel}
      </button>
    </div>
  </div>

  {#if expanded}
    <div class="panel">
      <div class="helper">
        Enter flags exactly as you would on the CLI, for example <code>--bootstrap-url google.com/pill --no-demand</code>.
      </div>
      {#if $URBIT_MODE && ownShip}
        <div class="helper warning">
          Saving rebuilds this ship and may temporarily disconnect the current GroundSeg session.
        </div>
      {/if}
      <textarea
        class:error={statusMessage.length > 0}
        bind:value={draft}
        disabled={isLoading}
        placeholder="--bootstrap-url google.com/pill --no-demand" />
      <div class="preview-label">Boot command preview</div>
      <pre>{preview}</pre>
      {#if statusMessage.length > 0}
        <div class="status-message">{statusMessage}</div>
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
  code {
    font-family: 'Source Code Pro', monospace;
    font-size: 90%;
  }
  .warning {
    color: #f1b96b;
  }
  textarea {
    min-height: 108px;
    resize: vertical;
    border-radius: 16px;
    color: var(--Gray-200, #ABBAAE);
    font-family: 'Source Code Pro', monospace;
    font-size: 18px;
    line-height: 28px;
    padding: 16px;
    border: 2px solid var(--text-card-color);
    background: var(--bg-card);
  }
  textarea.error {
    border-color: #d45151;
    color: #ffd4d4;
  }
  textarea:disabled {
    opacity: .6;
    pointer-events: none;
  }
  .preview-label {
    color: var(--text-card-color);
    font-family: Inter;
    font-size: 16px;
    font-weight: 300;
    letter-spacing: -0.96px;
  }
  pre {
    margin: 0;
    white-space: pre-wrap;
    word-break: break-word;
    border-radius: 16px;
    border: 1px solid #3E5142;
    background: #1E281F;
    color: #D9E2DB;
    padding: 16px;
    font-family: 'Source Code Pro', monospace;
    font-size: 15px;
    line-height: 24px;
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
    letter-spacing: -1.2px;
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
