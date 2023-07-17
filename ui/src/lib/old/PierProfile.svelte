<script>
	import { onMount } from 'svelte'
  import { structure } from '$lib/stores/websocket.js'

	import Sigil from '$lib/Sigil.svelte'
	import Clipboard from 'clipboard'

  import Fa from 'svelte-fa'
  import { faWrench, faHammer } from '@fortawesome/free-solid-svg-icons'

  export let name
  export let running
  export let code
  export let devMode
  export let click

  let copyPatp
  let clickedPatp = false

  $: rebuildInfo = ($structure?.urbits?.[name]?.container?.rebuild) || ""

  const rebuildContainer = () => {
    let payload = {
      "category": "urbits",
      "payload": {"patp": name, "module": "container", "action": "rebuild"}
    }
    //send($socket, $socketInfo, document.cookie, payload)
  }

  const patpClipboard = () => {
 		copyPatp = new Clipboard('#patp')
    copyPatp.on("success", ()=> {
    clickedPatp = true; setTimeout(()=> clickedPatp = false, 1000)})
  }

	onMount(()=> patpClipboard())

</script>

<div class="wrapper">
  <Sigil patp={name} size="80px" rad="12px" />

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
    {#if rebuildInfo.length < 1}
      <button class="rebuild" on:click={rebuildContainer}>
        <Fa icon={faHammer} size="1x" />
        <span class="rebuild-text">
          Rebuild
        </span>
      </button>
    {:else if rebuildInfo == "removing"}
      <div class="loading">Removing the container</div>
    {:else if rebuildInfo == "rebuilding"}
      <div class="loading">Rebuilding the container</div>
    {:else if rebuildInfo == "starting"}
      <div class="loading">Restarting the ship</div>
    {:else if rebuildInfo == "success"}
      <div class="loading success">Rebuild completed!</div>
    {:else if rebuildInfo.includes('failure')}
      <div class="loading failure">Error: {rebuildInfo.split('\n')[1]}</div>
    {/if}
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
  .rebuild {
    margin-top: 10px;
    cursor: pointer;
    color: inherit;
    height: 24px;
    background: #ffffff4d;
    font-size: 12px;
    padding: 2px 8px;
    border-radius: 4px;
    display: flex;
    gap: 8px;
    align-items: center;
  }
  .loading {
    animation: breathe 2s infinite;
    height: 24px;
    padding: 2px 0;
    line-height: 24px;
    font-size: 12px;
  }
  .success {
    color: lime;
    animation: none;
  }
  .failure {
    color: red;
    animation: none;
  }
</style>
