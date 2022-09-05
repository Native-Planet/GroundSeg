<script>
  import { api, scrollDown } from '$lib/api'
  import { page } from '$app/stores';
  import { onMount, onDestroy } from 'svelte'

  let log = $page.params.container,
    stream = [],
    shown = false
    

  onMount(() => {shown=true;getLog();})
  onDestroy(() => {shown=false;scrollDown.set(true);})

  const getLog = () => {
    if (shown) {
      const u = api + "/settings/logs"
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
<svelte:window on:wheel={()=>scrollDown.set(false)} />
<div>
{#each stream as s}
  <div class="content">{s}</div>
{/each}
<div id="jump"></div>
{#if !$scrollDown}
  <button class="latest" on:click={()=>{window.location.href="#jump";scrollDown.set(true)}}>Show Latest</button>
{/if}
</div>

<style>
  .content {
    margin-bottom: 4px;
    font-size: 12px;
    font-family:Consolas,Monaco,Lucida Console,Liberation Mono,DejaVu Sans Mono,Bitstream Vera Sans Mono,Courier New;
    width: 800px;
    padding: 0 20px 0 20px;
    max-width: calc(100vw - 40px);
  }
  .latest {
    position: sticky;
    bottom: 0;
    width: 100%;
    background: #0a0a0a8d;
    backdrop-filter: blur(30px);
    color: inherit;
    border: none;
    overflow: hidden;
    padding: 12px 0 12px 0;
  }
</style>
