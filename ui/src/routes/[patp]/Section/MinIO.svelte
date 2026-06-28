<script>
  // Style
  import "../theme.css"

  import Clipboard from 'clipboard'
  import { onMount } from 'svelte'
  import { toggleRustFSLink } from '$lib/stores/websocket'
  import CustomMinIODomain from './CustomMinIODomain.svelte'
  import { URBIT_MODE } from '$lib/stores/data'
  $: pfx = $URBIT_MODE ? "/apps/groundseg" : ""

  export let running
  export let minioAlias
  export let minioAliasMode = "local"
  export let patp
  export let minioUrl
  export let minioPwd
  export let minioLinked

  export let tToggleMinIOLink

  let copy
  let copied = false
  let showCustom = false
  $: s3Ready = (minioPwd?.length > 0) && (minioUrl != "#")

  onMount(()=>{
    copy = new Clipboard('#copy');
    copy.on("success", ()=> {
      copied = true;
      setTimeout(()=> copied = false, 1000)
    })
  })
</script>

<div>
  <div class="section-title">S3</div>
  <div class="wrapper">
    <button disabled={!s3Ready} id="copy" class="btn copy-btn" data-clipboard-text={minioPwd}>
      <img
        src={pfx+"/clipboard.svg"}
        alt="clipboard"
        width="24px"
        height="24px" />
      {#if copied}
        Copied!
      {:else}
        Copy S3 Password
      {/if}
    </button>
    <a href={minioUrl} target="_blank" class="btn">
      Settings
    </a>
    <button
      class="btn"
      on:click={()=>toggleRustFSLink(patp)}
      disabled={(tToggleMinIOLink == "linking") || !s3Ready || !running || (tToggleMinIOLink == "unlinking")}
      >
      {#if tToggleMinIOLink == "linking"}
        Linking..
      {:else if tToggleMinIOLink == "unlinking"}
        Unlinking..
      {:else if tToggleMinIOLink == "success"}
        S3 connected!
      {:else if tToggleMinIOLink == "unlink-success"}
        S3 disconnected!
      {:else}
        {minioLinked ? "Disconnect" : "Connect to Urbit"}
      {/if}
    </button>
    <div class="spacer"></div>
    <button
      class="btn domain-btn"
      class:active={showCustom}
      on:click={()=>showCustom = !showCustom}>
      {minioAlias.length > 0 ? "Modify custom domain" : "Set custom domain"}
    </button>
  </div>
  {#if showCustom}
    <!-- Custom S3 Domain -->
    <CustomMinIODomain {patp} {minioAlias} {minioAliasMode} on:done={()=>showCustom = false} />
  {/if}
</div>

<style>
  .wrapper {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 16px;
    margin: 16px 0;
  }
  .btn {
    color: #161D17;
    text-align: center;
    font-family: Inter;
    font-size: 20px;
    font-style: normal;
    font-weight: 300;
    line-height: 24px;
    letter-spacing: 0;
    padding: 16px;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    border-radius: 12px;
    background: var(--NP_White, #F8F8F6);
    cursor: pointer;
    flex: 0 0 auto;
    width: auto;
  }
  .btn:disabled {
    opacity: .6;
    pointer-events: none;
  }
  .copy-btn {
    min-width: 230px;
  }
  .domain-btn {
    background: #2C3A2E;
    color: white;
    padding: 16px 24px;
  }
  .spacer {
    flex: 1 1 24px;
    min-width: 0;
  }
  .active {
    background: var(--btn-secondary);
  }
</style>
