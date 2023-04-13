<script>
	import { onMount } from 'svelte'

	import { api } from '$lib/api'
	import Sigil from '$lib/Sigil.svelte'
	import Clipboard from 'clipboard'

  import Fa from 'svelte-fa'
  import { faWrench } from '@fortawesome/free-solid-svg-icons'

  export let name
  export let running
  export let code
  export let devMode
  export let click

  let copyPatp
  let clickedPatp = false

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
      <div class="status booting">{ devMode && !click ? "Boot Status: Unknown" : "Booting"}</div>
    {:else}
       <div class="status running">Running</div>
    {/if}

    <div
      on:click={copyPatp}
      data-clipboard-text={name}
      id="patp"
      class="patp">
      {#if devMode}
        <div class="dev-mode">
          <Fa icon={faWrench} size="1x"/>
        <div class="dev-text">dev mode</div>
        </div>
      {/if}
      {clickedPatp ? "copied!" : name}
    </div>
	</div>
</div>

<style>
  .wrapper {
    display: flex;
    gap: 20px;
    align-items: center;
		padding: 20px 0 10px 0;
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
    display: flex;
    gap: 8px;
    font-size: 16px;
    cursor: pointer;
    align-items: center;
  }
  .info {
		text-align: left;
  }
  .dev-mode {
    display: flex;
    gap: 8px;
    border-radius: 12px;
    background: #E6812F66;
    font-size: 10px;
    padding: 8px 12px;
    align-items: center;
  }
</style>
