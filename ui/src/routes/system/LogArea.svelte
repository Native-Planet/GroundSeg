<script>
  import { onMount, onDestroy } from 'svelte'
  import { logs, toggleLog } from '$lib/stores/websocket'
  export let type

  $: lines = ($logs[type]) || ""
  $: splitLines = lines.split("\n") || []

  onMount(()=>toggleLog(type,true))
  onDestroy(()=>toggleLog(type,false))
</script>
<div class="logarea">
  {#if (splitLines.length > 0)}
    {#each splitLines as ln}
      <div class="log-line">{ln}</div>
    {/each}
  {/if}
</div>

<style>
  .logarea {
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

</style>
