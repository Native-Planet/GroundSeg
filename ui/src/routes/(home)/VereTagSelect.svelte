<script>
  import { structure } from '$lib/stores/data'
  import { setVereTag } from '$lib/stores/websocket'

  export let patp
  export let urbitVersion = ""
  export let urbitImageTagOverride = ""
  export let versionServerVereTag = ""
  export let vereTags = []

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
  $: title = remoteMessage.length > 0
    ? remoteMessage
    : urbitImageTagOverride.length > 0
      ? `Vere image tag override: ${urbitImageTagOverride}`
      : `Vere image tag from version server: ${versionServerVereTag || urbitVersion || "current"}`
  $: status = isLoading ? "saving" : isSuccess ? "saved" : remoteMessage.length > 0 ? "error" : ""

  const handleChange = () => {
    if (draft !== urbitImageTagOverride) {
      setVereTag(patp, draft)
    }
  }
</script>

<div class="wrapper" class:override={urbitImageTagOverride.length > 0} class:error={remoteMessage.length > 0}>
  <div class="title">Vere</div>
  <div class="select-wrap" title={title}>
    <select
      aria-label="Vere image tag"
      bind:value={draft}
      disabled={isLoading}
      on:change={handleChange}>
      <option value="">server {versionServerVereTag || "current"}</option>
      {#each tags as tag}
        <option value={tag}>{tag}</option>
      {/each}
    </select>
  </div>
  {#if status.length > 0}
    <div class="status" class:error={remoteMessage.length > 0}>{status}</div>
  {/if}
</div>

<style>
  .wrapper {
    display: flex;
    align-items: center;
    gap: 5px;
    max-width: 292px;
  }
  .title,
  .status {
    color: #637061;
    font-family: var(--title-font);
    font-size: 12px;
    font-style: normal;
    font-weight: 700;
    line-height: normal;
    letter-spacing: 0;
    text-transform: uppercase;
  }
  .override > .title {
    color: var(--text-card-color);
  }
  .select-wrap {
    position: relative;
    width: 142px;
    height: 28px;
    flex-shrink: 0;
  }
  .select-wrap::after {
    content: "";
    position: absolute;
    right: 9px;
    top: 11px;
    width: 0;
    height: 0;
    border-left: 4px solid transparent;
    border-right: 4px solid transparent;
    border-top: 5px solid var(--text-card-color);
    pointer-events: none;
  }
  select {
    appearance: none;
    width: 100%;
    height: 100%;
    border: 0;
    border-radius: 4px 0px 0px 4px;
    background: var(--text-color, #313933);
    color: var(--text-card-color);
    cursor: pointer;
    font-family: var(--title-font);
    font-size: 12px;
    font-weight: 700;
    letter-spacing: 0;
    line-height: 28px;
    overflow: hidden;
    padding: 0 22px 0 8px;
    text-overflow: ellipsis;
    text-transform: uppercase;
    white-space: nowrap;
  }
  .override select {
    background: var(--Gray-400, #5C7060);
  }
  .error select {
    color: #ffd4d4;
  }
  select:disabled {
    cursor: default;
    opacity: .6;
  }
  .status {
    color: var(--Gray-300, #8FA393);
    font-size: 10px;
    letter-spacing: 0;
  }
  .status.error {
    color: #d45151;
  }
</style>
