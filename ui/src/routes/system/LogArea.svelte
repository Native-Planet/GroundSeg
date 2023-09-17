<script>
  import { onMount, onDestroy, beforeUpdate, afterUpdate } from 'svelte'
  import { logs, toggleLog } from '$lib/stores/websocket'
  import Clipboard from 'clipboard'

  export let type
  let div
	let autoscroll
  let copied = false

  $: lines = ($logs[type]) || ""
  $: splitLines = lines.split("\n") || []
  $: prettyLines = splitLines.map(str=>{
    try {
      let parsedJSON = JSON.parse(str) 
      return JSON.stringify(parsedJSON, null, 2)
    } catch {
      return str
    }
  })

  onMount(()=>{
    toggleLog(type,true)
    toLatest()
  })
  onDestroy(()=>toggleLog(type,false))
	beforeUpdate(() => {
		autoscroll = div && (div.offsetHeight + div.scrollTop) > (div.scrollHeight - 0);
	})
	afterUpdate(() => {
		if (autoscroll) div.scrollTo(0, div.scrollHeight);
	})

  const toLatest = () => {
    div.scrollTo(0, div.scrollHeight)
  }

  let copy = new Clipboard('#logs');
    copy.on("success", ()=> {
      copied = true;
      setTimeout(()=> copied = false, 1000)
    })
</script>

<div class="logarea">
  {#if copied}
    <div class="copy">copied!</div>
  {:else}
    <img id="logs" data-clipboard-text={lines} class="copy" src="/clipboard.svg" size="25px" alt="copy icon" />
  {/if}
  {#if !autoscroll}
    <button on:click={toLatest} class="latest">Latest</button>
  {/if}
  <div class="log-wrapper" bind:this={div}>
    {#if (prettyLines.length > 0)}
      {#each prettyLines as ln}
        <pre class="log-line">{ln}</pre>
      {/each}
    {/if}
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
  .log-line {
    display: flex;
    gap: 20px;
    color: #000;
    leading-trim: both;
    text-edge: cap;
    font-family: var(--log-font, Source Code Pro);
    font-size: 16px;
    font-style: normal;
    font-weight: 400;
    line-height: 20px; /* 125% */
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
