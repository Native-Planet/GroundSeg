<script>
  // Style
  import "../theme.css"
  import Clipboard from 'clipboard'
  import { onMount, createEventDispatcher } from 'svelte'
  import CustomUrbitDomain from './CustomUrbitDomain.svelte'
  import { toggleUrbitAlias } from '$lib/stores/websocket'
  import { page } from '$app/stores'

  import Fa from 'svelte-fa'
  import { faRepeat } from '@fortawesome/free-solid-svg-icons'

  export let urbitAlias
  export let showUrbAlias
  export let patp
  export let url = "#"
  export let lusCode = ""
  export let running = false
  export let startramRegistered = false
  export let remote = false

  let urlType;
  $: {
    try {
      urlType = new URL(url);
    } catch (error) {
      urlType = null;
    }
  }
  $: urlStripped = urlType == null ? url : `${urlType?.hostname}`
  $: urlFixed = urlStripped == null ? url : remote ? url : (urlStripped == url) ? url : "http://" + $page.url.hostname + ":" + urlType.port 
  $: displayedUrl = (showUrbAlias ? "https://"+urbitAlias : urlFixed)

  const dispatch = createEventDispatcher()
  let copied = false
  let copy
  let showCustom = false

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
        src="/clipboard.svg"
        width="24px"
        height="24px" />
      {#if copied}
        Copied!
      {:else}
        Access Key
      {/if}
    </button>
    <a href={displayedUrl} class:disabled={!running} target="_blank" class="btn">
      {#if showUrbAlias && (urbitAlias.length > 0)}
        Custom 
      {/if}
      URL ↗ 
    </a>
    <div class="spacer"></div>
    {#if urbitAlias.length > 0}
      <button class="btn domain-btn" on:click={()=>toggleUrbitAlias(patp)}><Fa icon={faRepeat} size="1x" /> Switch URL</button>
    {/if}
    <button
      disabled={!startramRegistered}
      class="btn domain-btn"
      class:active={showCustom}
      on:click={()=>showCustom = !showCustom}>
      {urbitAlias.length > 0 ? "Modify" : "Set"} Custom Urbit Domain
    </button>
  </div>
  {#if showCustom}
    <!-- Custom Urbit Domain -->
    <CustomUrbitDomain {urbitAlias} {patp} on:done={()=>showCustom = false} />
  {/if}
</div>

<style>
  .section-title-wrapper {
    display: flex; 
    align-items: center;
    gap: 16px;
  }
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
