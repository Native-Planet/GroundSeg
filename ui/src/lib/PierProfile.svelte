<script>
	import { onMount } from 'svelte'

	import { api } from '$lib/api'
	import Sigil from '$lib/Sigil.svelte'
	import Clipboard from 'clipboard'

	export let name, running, code

	let copyPatp, clickedPatp = false

	onMount(()=> {
 		copyPatp = new Clipboard('#patp')
    copyPatp.on("success", ()=> {
    clickedPatp = true; setTimeout(()=> clickedPatp = false, 1000)})
	})

</script>

<div class="wrapper">
	<Sigil patp={name} size="72px" rad="12px" />

	<div class="info">
    {#if !running}
      <div class="status">Stopped</div>
    {:else if code == null}
      <div class="status loading">Loading...</div>
    {:else if code.length != 27}
      <div class="status booting">Booting</div>
    {:else}
       <div class="status running">Running</div>
    {/if}

    <div
      on:click={copyPatp}
      data-clipboard-text={name}
      id="patp"
      class="patp">
      {clickedPatp ? "copied!" : name}
    </div>
	</div>
</div>

<style>

  .wrapper {
    display: flex;
    gap: 20px;
    align-items: center;
		padding: 20px 0 20px 0;
  }
  .status {
    opacity: .8;
    font-weight: 400;
    font-size: .8em;
    padding-bottom: 6px;
    color: red;
  }
  .loading {
    color: white;
  }
  .booting {
    color: orange;
  }
  .running {
    color: lime;
  }
  .patp {
    font-size: 16px;
    cursor: pointer;
  }
  .info {
		text-align: left;
  }
</style>
