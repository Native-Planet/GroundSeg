<script>
  // Style
  import "../theme.css"

  import Clipboard from 'clipboard'
  import { onMount, createEventDispatcher } from 'svelte'
  import { toggleMinIOLink } from '$lib/stores/websocket'
  import CustomMinIODomain from './CustomMinIODomain.svelte'

  export let startramRunning
  export let patp
  export let minioUrl
  export let minioPwd
  export let minioLinked

  export let tToggleMinIOLink

  let copy
  let copied = false
  let showCustom = false

  onMount(()=>{
    copy = new Clipboard('#copy');
    copy.on("success", ()=> {
      copied = true;
      setTimeout(()=> copied = false, 1000)
    })
  })
</script>

<div>
  <div class="section-title">MinIO</div>
  <div class="wrapper">
    <button disabled={!startramRunning} id="copy" class="btn copy-btn" data-clipboard-text={minioPwd}>
      <img
        src="/clipboard.svg"
        width="24px"
        height="24px" />
      {#if copied}
        Copied!
      {:else}
        Copy MinIO Password
      {/if}
    </button>
    <a href={minioUrl} target="_blank" class="btn">
      Settings
    </a>
    <button
      class="btn"
      on:click={()=>toggleMinIOLink(patp)}
      disabled={(tToggleMinIOLink == "linking") || !startramRunning}
      >
      {#if tToggleMinIOLink == "success"}
        MinIO connected!
      {:else}
        {minioLinked ? "Disconnect" : "Connect to Urbit"}
      {/if}
    </button>
    <div class="spacer"></div>
    <button disabled={!startramRunning} class="btn domain-btn" class:active={showCustom} on:click={()=>showCustom = !showCustom}>Set Custom MinIO Domain</button>
  </div>
  {#if showCustom}
    <!-- Custom MinIO Domain -->
    <CustomMinIODomain {patp} />
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
  .btn:disabled {
    opacity: .6;
    pointer-events: none;
  }
  .copy-btn {
    width: 285px;
  }
  .domain-btn {
    background: #2C3A2E;
    color: white;
    padding: 0 48px;
  }
  .spacer {
    flex: 1;
  }
  .active {
    background: var(--btn-secondary);
  }
</style>
