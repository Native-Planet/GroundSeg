<script>
  // Style
  import "../theme.css"
  import Clipboard from 'clipboard'
  import { onMount, createEventDispatcher } from 'svelte'
  import CustomUrbitDomain from './CustomUrbitDomain.svelte'
  import { toggleUrbitAlias } from '#lib/stores/websocket.js'
  import { page } from '$app/state'

  import Fa from 'svelte-fa'
  import { faRepeat } from '@fortawesome/free-solid-svg-icons'
  import { URBIT_MODE } from '#lib/stores/data.js'
  let pfx = $derived($URBIT_MODE ? "/apps/groundseg" : "")

  /**
   * @typedef {Object} Props
   * @property {any} urbitAlias
   * @property {any} showUrbAlias
   * @property {any} patp
   * @property {string} [url]
   * @property {string} [lusCode]
   * @property {boolean} [running]
   * @property {boolean} [startramRegistered]
   * @property {boolean} [remote]
   */

  /** @type {Props} */
  let {
    urbitAlias,
    showUrbAlias,
    patp,
    url = "#",
    lusCode = "",
    running = false,
    startramRegistered = false,
    remote = false
  } = $props();

  let urlType = $derived.by(() => {
    try {
      return new URL(url);
    } catch (error) {
      return null;
    }
  });
  let urlStripped = $derived(urlType == null ? url : `${urlType?.hostname}`)
  let urlFixed = $derived(urlStripped == null ? url : remote ? url : (urlStripped == url) ? url : "http://" + page.url.hostname + ":" + urlType.port)
  let displayedUrl = ($derived(showUrbAlias && remote ? "https://"+urbitAlias : urlFixed))

  const dispatch = createEventDispatcher()
  let copied = $state(false)
  let copy
  let showCustom = $state(false)

  onMount(()=>{
    copy = new Clipboard('#lus-code');
    copy.on("success", ()=> {
      copied = true;
      setTimeout(()=> copied = false, 1000)
    })
  })
</script>

<div>
  <div class="section-title">Urbit Information</div>
  <div class="wrapper">
    <button disabled={!running || lusCode.length < 27} id="lus-code" class="btn" data-clipboard-text={lusCode}>
      <img
        src={pfx+"/clipboard.svg"}
        alt="clipboard"
        width="24px"
        height="24px" />
      {#if copied}
        Copied!
      {:else}
        Access Key
      {/if}
    </button>
    <a href={displayedUrl} class:disabled={!running} target="_blank" class="btn">
      {showUrbAlias && (urbitAlias.length > 0) && remote ? "Custom" : ""} URL ↗
    </a>
    <div class="spacer"></div>
    {#if urbitAlias.length > 0 && remote}
      <button class="btn domain-btn" onclick={()=>toggleUrbitAlias(patp)}><Fa icon={faRepeat} size="1x" /> Switch URL</button>
    {/if}
    <button
      disabled={!startramRegistered}
      class="btn domain-btn"
      class:active={showCustom}
      onclick={()=>showCustom = !showCustom}>
      {urbitAlias.length > 0 ? "Modify" : "Set"} Custom Urbit Domain
    </button>
  </div>
  {#if showCustom}
    <!-- Custom Urbit Domain -->
    <CustomUrbitDomain {urbitAlias} {patp} on:done={()=>showCustom = false} />
  {/if}
</div>

<style>
  .wrapper {
    display: flex;
    gap: 16px;
    margin: 16px 0;
  }
  .btn {
    color: #161D17;
    text-align: right;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    line-height: 24px; /* 100% */
    letter-spacing: -1.44px;
    padding: 16px;
    display: flex;
    align-items: center;
    gap: 8px;
    border-radius: 12px;
    background: var(--NP_White, #F8F8F6);
    cursor: pointer;
  }
  .spacer {
    flex: 1;
  }
  /*
  .what {
    width: 20px;
    height: 20px;
    text-align: center;
    border: 1px solid #FFF;
    border-radius: 50%;
    cursor: pointer;
    font-size: 16px;
  }
  .what:hover {
    opacity: .2;
  }
  */
  .domain-btn {
    background: #2C3A2E;
    color: white;
    padding: 0 48px;
  }
  .btn:disabled {
    opacity: .6;
    pointer-events: none;
  }
  .disabled {
    opacity: .6;
    pointer-events: none;
  }
  .active {
    background: var(--btn-secondary);
  }
</style>
