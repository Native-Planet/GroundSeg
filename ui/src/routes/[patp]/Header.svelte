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
  $: displayedVereTag = urbitImageTagOverride || defaultVereTag || "current"
  $: versionOptions = [...new Set([defaultVereTag, urbitImageTagOverride, ...vereTags].filter(Boolean))]
    .sort((a, b) => b.localeCompare(a, undefined, { numeric: true, sensitivity: "base" }))
    .map(tag => ({ tag, value: tag == defaultVereTag ? "" : tag }))
  $: versionTitle = vereError || (urbitImageTagOverride
    ? `Vere image tag override: ${urbitImageTagOverride}`
    : `Vere image tag: ${defaultVereTag || "current"}`)

  $: chars = (patp.replace(/-/g,"").length) || 0
  $: shipClass = (chars == 3 ? "GALAXY" : chars == 6 ? "STAR" : chars == 12 ? "PLANET" : chars > 12 ? "MOON" : "UNKNOWN") || "ERROR"

  let copy
  let copied = false
  let draftVereTag = urbitImageTagOverride
  let lastVereTag = urbitImageTagOverride
  let versionMenu
  let versionMenuOpen = false

  $: if (urbitImageTagOverride !== lastVereTag) {
    draftVereTag = urbitImageTagOverride
    lastVereTag = urbitImageTagOverride
  }

  const toggleVersionMenu = () => {
    if (!isSavingVere && versionOptions.length > 0) {
      versionMenuOpen = !versionMenuOpen
    }
  }

  const selectVereTag = value => {
    versionMenuOpen = false
    draftVereTag = value
    if (value !== urbitImageTagOverride) {
      setVereTag(patp, value)
    }
  }

  onMount(()=>{
    copy = new Clipboard('#patp');
    copy.on("success", ()=> {
      copied = true;
      setTimeout(()=> copied = false, 1000)
    })

    const closeVersionMenu = event => {
      if (versionMenu && !versionMenu.contains(event.target)) {
        versionMenuOpen = false
      }
    }
    document.addEventListener('click', closeVersionMenu)
    return () => document.removeEventListener('click', closeVersionMenu)
  })

</script>

<div class="header">
  <div class="patp-wrapper">
    <div class="ship-class">{shipClass}
      <span
        class="version-control"
        class:open={versionMenuOpen}
        class:override={urbitImageTagOverride.length > 0}
        class:error={vereError.length > 0}
        bind:this={versionMenu}>
        <button
          type="button"
          disabled={isSavingVere || versionOptions.length == 0}
          title={versionTitle}
          aria-label="Vere image tag"
          aria-expanded={versionMenuOpen}
          on:click|stopPropagation={toggleVersionMenu}>
          {displayedVereTag.toUpperCase()}
        </button>
        {#if versionMenuOpen}
          <div class="version-menu" role="listbox" aria-label="Vere image tags">
            {#each versionOptions as option}
              <button
                type="button"
                class:active={option.value == draftVereTag}
                class:default={option.value == ""}
                on:click={() => selectVereTag(option.value)}>
                {option.tag.toUpperCase()}
              </button>
            {/each}
          </div>
        {/if}
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
    z-index: 10;
  }
  .version-control::after {
    content: "";
    position: absolute;
    right: 7px;
    top: 12px;
    width: 0;
    height: 0;
    border-left: 3px solid transparent;
    border-right: 3px solid transparent;
    border-top: 4px solid var(--text-card-color);
    pointer-events: none;
  }
  .version-control.open::after {
    transform: rotate(180deg);
  }
  .version-control > button {
    background: transparent;
    border: 1px solid var(--Gray-400, #5C7060);
    border-radius: 4px;
    color: var(--text-card-color);
    cursor: pointer;
    font-family: var(--title-font);
    font-size: 18px;
    font-weight: 700;
    height: 30px;
    letter-spacing: 0;
    line-height: 28px;
    max-width: 150px;
    min-width: 74px;
    overflow: hidden;
    padding: 0 20px 0 7px;
    text-overflow: ellipsis;
    text-transform: uppercase;
    white-space: nowrap;
  }
  .version-control.override > button {
    background: var(--Gray-400, #5C7060);
  }
  .version-control.error > button {
    border-color: #d45151;
    color: #ffd4d4;
  }
  .version-control > button:disabled {
    cursor: default;
    opacity: .6;
  }
  .version-menu {
    position: absolute;
    top: 34px;
    left: 0;
    width: 162px;
    max-height: 300px;
    overflow-y: auto;
    border: 1px solid var(--Gray-400, #5C7060);
    border-radius: 6px;
    background: var(--bg-modal, #F5F1E8);
    box-shadow: 0 10px 24px rgba(0, 0, 0, .22);
    padding: 4px;
  }
  .version-menu button {
    display: block;
    width: 100%;
    height: 29px;
    border: 0;
    border-radius: 4px;
    background: transparent;
    color: var(--NP_Black, #313933);
    cursor: pointer;
    font-family: var(--title-font);
    font-size: 18px;
    font-weight: 700;
    letter-spacing: 0;
    overflow: hidden;
    padding: 0 8px;
    text-align: left;
    text-overflow: ellipsis;
    text-transform: uppercase;
    white-space: nowrap;
  }
  .version-menu button:hover,
  .version-menu button.active {
    background: var(--Gray-400, #5C7060);
    color: #fff;
  }
  .version-menu button.default {
    border-bottom: 1px solid rgba(49, 57, 51, .18);
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
