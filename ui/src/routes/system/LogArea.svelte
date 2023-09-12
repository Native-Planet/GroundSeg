<script>
  import { onMount, onDestroy, beforeUpdate, afterUpdate } from 'svelte'
  import { logs, toggleLog } from '$lib/stores/websocket'
  export let type
  let div
	let autoscroll

  $: lines = ($logs[type]) || ""
  $: splitLines = lines.split("\n") || []

  onMount(()=>toggleLog(type,true))
  onDestroy(()=>toggleLog(type,false))
	beforeUpdate(() => {
		autoscroll = div && (div.offsetHeight + div.scrollTop) > (div.scrollHeight - 0);
	})
	afterUpdate(() => {
		if (autoscroll) div.scrollTo(0, div.scrollHeight);
	})

  const toLatest = () => div.scrollTo(0, div.scrollHeight)


</script>
<div class="logarea" bind:this={div}>
  {#if !autoscroll}
    <button on:click={toLatest} class="latest">Latest</button>
  {/if}
  <div class="log-wrapper">
    {#if (splitLines.length > 0)}
      {#each splitLines as ln}
        <div class="log-line">{ln}</div>
      {/each}
      <div class="log-line"></div>
      <div class="log-line"></div>
      <div class="log-line"></div>
    {/if}
  </div>
</div>

<style>
  .logarea::-webkit-scrollbar {display: none;}
  .logarea {
    position: relative;
    background: var(--bg-modal);
    width: calc(100vw - (48px * 2) - (24px * 2) - 15px);
    border-radius: 16px;
    margin-top: 32px;
    padding: 24px;
    overflow-y: scroll;
    margin-bottom: 32px;
    height: 75%;
  }
  .log-line {
    display: flex;
    gap: 20px;
    color: #000;
    leading-trim: both;
    text-edge: cap;
    font-family: Source Code Pro;
    font-size: 16px;
    font-style: normal;
    font-weight: 400;
    line-height: 20px; /* 125% */
    letter-spacing: -0.96px;
  }
  .latest {
    position: fixed;
    bottom: 24px;
    background: var(--btn-secondary);
    right: 48px;
    width: 64px;
    line-height: 48px;
    height: 48px;
    font-size: 12px;
    color: var(--text-card-color);
    border-radius: 16px 0 16px 0;
  }

</style>
