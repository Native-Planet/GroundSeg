<script>
  import Clipboard from 'clipboard'
  import { onMount } from 'svelte'
  import { setVereTag } from '$lib/stores/websocket'
  import { structure } from '$lib/stores/data'
  import { devShipClass } from '$lib/stores/devclass'
  export let patp

  $: ship = ($structure?.urbits?.[patp]?.info) || {}
  $: memUsage = (ship?.memUsage) || 0
  $: diskUsage = (ship?.diskUsage) || 0
  $: loom = (ship?.loomSize) || 0
  $: loomActual = 2 ** loom / (1024 * 1024)
  $: vere = (ship?.vere) || ""
  $: urbitVersion = (ship?.urbitVersion) || ""
  $: urbitImageTagOverride = (ship?.urbitImageTagOverride) || ""
  $: versionServerVereTag = (ship?.versionServerVereTag) || ""
  $: vereTags = (ship?.vereTags) || []
  $: tVereTag = ($structure?.urbits?.[patp]?.transition?.vereTag) || ""
  $: isSavingVere = tVereTag == "loading"
  $: vereError = tVereTag.length > 0 && tVereTag != "loading" && tVereTag != "success" ? tVereTag : ""
  $: defaultVereTag = versionServerVereTag || urbitVersion || vere
  $: selectableVereTags = [...new Set([urbitImageTagOverride, ...vereTags].filter(Boolean))]
    .filter(tag => tag != defaultVereTag)
  $: versionTitle = vereError || (urbitImageTagOverride
    ? `Vere image tag override: ${urbitImageTagOverride}`
    : `Vere image tag: ${defaultVereTag || "current"}`)

  $: chars = (patp.replace(/-/g,"").length) || 0
  $: shipClass = (chars == 3 ? "GALAXY" : chars == 6 ? "STAR" : chars == 12 ? "PLANET" : chars > 12 ? "MOON" : "UNKNOWN") || "ERROR"

  let copy
  let copied = false
  let draftVereTag = urbitImageTagOverride
  let lastVereTag = urbitImageTagOverride

  $: if (urbitImageTagOverride !== lastVereTag) {
    draftVereTag = urbitImageTagOverride
    lastVereTag = urbitImageTagOverride
  }

  const changeVereTag = () => {
    if (draftVereTag !== urbitImageTagOverride) {
      setVereTag(patp, draftVereTag)
    }
  }

  onMount(()=>{
    copy = new Clipboard('#patp');
    copy.on("success", ()=> {
      copied = true;
      setTimeout(()=> copied = false, 1000)
    })
  })

</script>

<div class="header">
  <div class="patp-wrapper">
    <div class="ship-class">{shipClass}
      <span class="version-control" class:override={urbitImageTagOverride.length > 0} class:error={vereError.length > 0}>
        <select
          bind:value={draftVereTag}
          disabled={isSavingVere}
          title={versionTitle}
          aria-label="Vere image tag"
          on:change={changeVereTag}>
          <option value="">{(defaultVereTag || "current").toUpperCase()}</option>
          {#each selectableVereTags as tag}
            <option value={tag}>{tag.toUpperCase()}</option>
          {/each}
        </select>
      </span>
      {#if isSavingVere}
        <sup class="version-status">SAVING</sup>
      {:else if tVereTag == "success"}
        <sup class="version-status">SAVED</sup>
      {:else if vereError.length > 0}
        <sup class="version-status error">ERROR</sup>
      {/if}
    </div>
    <div class="patp" id="patp" data-clipboard-text={patp}>
      {#if copied}
        COPIED!
      {:else}
        <!-- dev modification -->
        {#if $devShipClass == "star"}
          {patp.slice(0,6).toUpperCase()}
        {:else if $devShipClass == "planet"}
          {patp.slice(0,13).toUpperCase()}
        {:else}
          {patp.toUpperCase()}
        {/if}
      {/if}
    </div>
  </div>
  <div class="settings-wrapper">
    <div class="settings">
      <div class="settings-text">RAM</div>
      <div class="settings-val">{parseInt(memUsage/(1024*1024))} MB / {loomActual} MB</div>
    </div>
    <div class="settings">
      <div class="settings-text">DISK</div>
      <div class="settings-val">{(diskUsage/(1024*1024)).toFixed(2)} MB</div>
    </div>
  </div>
</div>

<style>
  .header {
    background-color: var(--bg-card);
    color: var(--text-card-color);
    position: absolute;
    height: 150px;
    width: calc(1173px - 150px);
    left: 150px;
    border-radius: 16px 16px 0 0;
    position: relative;
  }
  .patp-wrapper {
    background: var(--fg-card);
    position: absolute;
    top: 16px;
    left: 16px;
    width: 567px;
    height: 134px;
    border-radius: 16px;
  }
  .ship-class {
    font-family: var(--title-font);
    margin: 28px 0 12px 20px;
    font-size: 18px;
  }
  .version-control {
    display: inline-flex;
    position: relative;
    vertical-align: super;
    margin-left: 5px;
  }
  .version-control::after {
    content: "";
    position: absolute;
    right: 5px;
    top: 9px;
    width: 0;
    height: 0;
    border-left: 3px solid transparent;
    border-right: 3px solid transparent;
    border-top: 4px solid var(--text-card-color);
    pointer-events: none;
  }
  .version-control select {
    appearance: none;
    background: transparent;
    border: 1px solid var(--Gray-400, #5C7060);
    border-radius: 4px;
    color: var(--text-card-color);
    cursor: pointer;
    font-family: var(--title-font);
    font-size: 11px;
    font-weight: 700;
    height: 22px;
    letter-spacing: 0;
    max-width: 134px;
    min-width: 58px;
    overflow: hidden;
    padding: 0 15px 0 5px;
    text-overflow: ellipsis;
    text-transform: uppercase;
    white-space: nowrap;
  }
  .version-control.override select {
    background: var(--Gray-400, #5C7060);
  }
  .version-control.error select {
    border-color: #d45151;
    color: #ffd4d4;
  }
  .version-control select:disabled {
    cursor: default;
    opacity: .6;
  }
  .version-status {
    margin-left: 5px;
    color: var(--Gray-300, #8FA393);
    font-size: 10px;
    letter-spacing: 0;
  }
  .version-status.error {
    color: #d45151;
  }
  .patp {
    cursor: pointer;
    font-family: var(--title-font);
    margin-left: 20px;
    font-size: 32px;
  }
  .settings-wrapper {
    background: var(--fg-card);
    font-family: var(--title-font);
    position: absolute;
    display: flex;
    flex-direction: column;
    gap: 8px;
    width: 320px;
    padding: 24px 32px;
    right: 0;
    border-radius: 0px 16px;
  }
  .settings {
    display: flex;
    font-size: 24px;
  }
  .settings-text {
    flex: 1;
  }
</style>
