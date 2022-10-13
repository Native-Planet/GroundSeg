<script>
  import { api, scrollDown } from '$lib/api'
  import { logs } from '$lib/components'
  import { onMount, onDestroy } from 'svelte'

  export let log, maxHeightOffset = 0

  let stream = null, shown = false

  onMount(() => {shown=true;getLog();})
  onDestroy(() => {shown=false;scrollDown.set(true);})

  const handleScroll = e => {
    if ((e.deltaY < 0) && (Array.isArray(stream))) {
      scrollDown.set(false)
    }
  }

  const getLog = () => {
    if (shown) {
      const u = $api + "/settings/logs"
      const f = new FormData()
      f.append('logs', log)
      fetch(u, {method: 'POST', body: f})
        .then(r => r.json()).then(d => {
          stream = d.split("\n")
        })
      if ($scrollDown) {
        window.location.href="#jump"
      }
      setTimeout(getLog, 1000)
    }
  }
</script>

<svelte:window on:wheel={handleScroll} />

{#if Array.isArray(stream)}
  <!-- detect mousewheel event -->

  <div class="wrapper">

    <svelte:component this={logs.logo} />

    <!-- stream of selected log -->
    <div class="logs" style="max-height: calc(80vh - 92px - {maxHeightOffset}px;">
      {#each stream as s}<div class="content">{s}</div>{/each}
      <div id="jump"></div>
    </div>

    <!-- send to bottom of screen -->
    {#if !$scrollDown}
      <button class="latest" on:click={()=>{window.location.href="#jump";scrollDown.set(true)}}>Show Latest</button>
    {/if}

  </div>
{:else}
  <div class="wrapper">

    <svelte:component this={logs.logo} />

    <div class="blurred"></div>

  </div>

{/if}

<style>
  @keyframes breathe {
    0% {opacity: .6}
    50% {opacity: 0}
    100% {opacity: .6}
  }
  .blurred {
    height: 40vh;
    background: #ffffff4d;
    width: calc(100% - 40px);
    margin: 20px;
    border-radius: 15px;
    filter: blur(20px);
    animation: breathe 2s infinite;
  }
  .wrapper {
    position: relative;
  }
  .logs {
    overflow: auto;
    margin-top: 20px;
    background: none;
    -ms-overflow-style: none;
    scrollbar-width: none;
  }
  .logs::-webkit-scrollbar {
    display: none;
  }

  .content {
    margin-bottom: 4px;
    font-size: 12px;
    font-family:Consolas,Monaco,Lucida Console,Liberation Mono,DejaVu Sans Mono,Bitstream Vera Sans Mono,Courier New;
    width: calc(100% - 40px);
    padding: 0 20px 0 20px;
  }
  .latest {
    position: absolute;
    bottom: 0;
    width: 100%;
    background: #0a0a0a8d;
    backdrop-filter: blur(20px);
    color: inherit;
    border: none;
    padding: 12px 0 12px 0;
    border-radius: 15px;
  }
</style>
