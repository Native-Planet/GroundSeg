<script>
  import { api, currentLog } from '$lib/api'
  import { onMount, onDestroy, beforeUpdate, afterUpdate } from 'svelte'

  export let container, maxHeight
  let div
	let autoscroll
  let shown = false

	beforeUpdate(() => {
		autoscroll = div && (div.offsetHeight + div.scrollTop) > (div.scrollHeight - 20);
	})

	afterUpdate(() => {
		if (autoscroll) div.scrollTo(0, div.scrollHeight);
	})
  onMount(() => {
    shown = true
    getLog()
  })
  onDestroy(() => {
    shown = false
    currentLog.set({'container':'','log':[]})
  })

  const toLatest = () => div.scrollTo(0, div.scrollHeight)
  const getLog = () => {
    if (container == '') {
      setTimeout(getLog, 1000) 
    } else {
      let module = 'logs'
      if (shown) {
        if ($currentLog.container != container) {
          currentLog.set({'container':'','log':[]})        
        }
  	    fetch($api + '/system?module=' + module, {
			    method: 'POST',
          credentials: "include",
			    headers: {'Content-Type': 'application/json'},
  			  body: JSON.stringify({'action':'view','container':container,'haveLine':$currentLog.log.length})
	      })
        .then(r => r.json())
        .then(d => {
          if (d == 404) {
            window.location.href = '/login'
          } else {
          currentLog.update( s => {
            s['container'] = container
            s['log'] = s['log'].concat(d)
            return s
          })
          setTimeout(getLog, 1000)
          }
        })
    }}
  }

</script>

{#if shown}
  <div class="logs-wrapper" bind:this={div} style="max-height:{maxHeight}">
  <!-- Print log array -->
  {#each $currentLog.log as ln}
    {#if ln.length > 0}
      <code>{ln}</code><br>
    {/if}
  {/each}

  <!-- Jump to latest -->
  {#if !autoscroll}<button on:click={toLatest} class="latest">Show Latest</button>{/if}
</div>
{/if}

<style>
	.logs-wrapper::-webkit-scrollbar {display: none;}
  .logs-wrapper {
    margin-top: 12px;
    overflow: auto;
  }
  code {
    font-size: 12px;
    font-weight: 300;
  }
  .latest {
    position: sticky;
    bottom: 0;
    width: 100%;
    background: #0404048D;
    padding: 12px;
    backdrop-filter: blur(20px);
    color: inherit;
    border: none;
    border-radius: 15px;
  }
</style>
<!--
<svelte:window on:wheel={handleScroll} />

{#if Array.isArray(stream)}
  <!-- detect mousewheel event --

  <div class="wrapper">

    <svelte:component this={logs.logo} />

    <!-- stream of selected log --
    <div class="logs" style="max-height: calc(80vh - 92px - {maxHeightOffset}px;">
      {#each stream as s}<div class="content">{s}</div>{/each}
      <div id="jump"></div>
    </div>

    <!-- send to bottom of screen --
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
-->
