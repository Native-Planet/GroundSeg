<script>
  import { page } from '$app/stores'
  import { onMount, onDestroy, beforeUpdate, afterUpdate } from 'svelte'
  import { URBIT_MODE } from '$lib/stores/data'
  import { wsPort } from '$lib/stores/websocket'
  import { logs, connect, disconnect } from '$lib/stores/logsocket'
  import Clipboard from 'clipboard'

  export let type
  let copied = false
  let latest = true

  $: lines = ($logs[type]) || []

  onMount(()=> {
    const hostname = $page.url.hostname
    if (!$URBIT_MODE) {
      connect("ws://" + hostname + ":" + $wsPort + "/logs", type)
    }
  })
  onDestroy(()=>disconnect(type))
  const toLatest = () => {
    window.location.href = "#latest"
    latest = true
  }

  let copy = new Clipboard('#logs');
    copy.on("success", ()=> {
      copied = true;
      setTimeout(()=> copied = false, 1000)
    })

  afterUpdate(()=>{
    if (latest) {
      toLatest()
    }
  })
  const handleWheel = e => {
    if (e.deltaY < 0) {
      latest = false   
    }
  }
</script>

<div class="logarea">
  {#if copied}
    <div class="copy">copied!</div>
  {:else}
    <img id="logs" data-clipboard-text={lines.map(obj => JSON.stringify(obj, null, 2)).join('\n')} class="copy" src="/clipboard.svg" size="25px" alt="copy icon" />
  {/if}
  {#if !latest}
    <button on:click={toLatest} class="latest">Latest</button>
  {/if}
  <div class="log-wrapper" on:wheel={handleWheel}>
    {#if (lines.length > 0)}{#each lines as ln}
      <pre class="log-line">
        {type == "system" ? JSON.stringify(ln,null,2) : ln}
      </pre>
    {/each}{/if}
    <div id="latest"></div>
  </div>
</div>

<style>
  .logarea {
    position: relative;
    background: var(--text-card-color);
    width: calc(100vw - (48px * 2) - (24px * 2) - 15px);
    border-radius: 16px;
    margin-top: 32px;
    height: 75%;
    padding: 24px;
  }
  .log-wrapper::-webkit-scrollbar {display: none;}
  .log-wrapper {
    overflow-y: scroll;
    height: 100%;
  }
  pre {
    margin: 0;
  }
  .log-line {
    display: flex;
    color: #000;
    font-family: var(--log-font, Source Code Pro);
    font-size: 16px;
    font-style: normal;
    font-weight: 400;
    letter-spacing: -0.96px;
  }
  .latest {
    position: absolute;
    bottom: 0;
    right: 0;
    background: var(--btn-secondary);
    width: 64px;
    line-height: 48px;
    height: 48px;
    font-size: 12px;
    color: var(--text-card-color);
    border-radius: 16px 0 16px 0;
    cursor: pointer;
  }
  .copy {
    position: absolute;
    right: 0px;
    top: 0px;
    background: #DDE3DF;
    border-radius: 0 16px 0 16px;
    padding: 16px;
    font-size: 14px;
    cursor: pointer;
  }
</style>
