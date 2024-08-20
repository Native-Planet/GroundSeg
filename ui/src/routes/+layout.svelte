<script>
  // Urbit
  import { broadcast, subscribe } from '$lib/stores/urbit.js'
  // Svelte
  import { onMount } from 'svelte'
  import { page } from '$app/stores'

  // Websocket
  import { wsPort, connect } from '$lib/stores/websocket'
  import { firstLoad, URBIT_MODE, DEV_PANEL } from '$lib/stores/data'
  import { wide } from '$lib/stores/display'

  import Redirector from './Redirector.svelte'
  import ApiSpinner from './ApiSpinner.svelte'
  import KeepAlive from './KeepAlive.svelte'
  import FirstLoad from './FirstLoad.svelte'

  // Style
  import "../theme.css"

  // Dev panel
  import { 
    devStartramReminder,
    devStartramReminderToggle,
    printMounts,
    devBackupTlon
  } from '$lib/stores/websocket'


  const isUrbitMode = process.env.GS_URBIT_MODE;
  const showDevPanel = process.env.GS_DEV_PANEL;

  onMount(()=> {
    URBIT_MODE.set(isUrbitMode)
    DEV_PANEL.set(showDevPanel)
    const hostname = $page.url.hostname
    if ($URBIT_MODE) {
      subscribe(window.ship)
    } else {
      connect("ws://" + hostname + ":" + $wsPort + "/ws")
    }
  })

	const vert = (h,w) => {
	  let r = h / w
    let d = false
		if ( r > 1) { d = true }
		wide.set(!d)	
	}

</script>

<!--svelte:window bind:innerWidth bind:innerHeight /-->
<Redirector />
{#if $firstLoad}
  <FirstLoad />
{:else}
  <slot/>
  <ApiSpinner />
{/if}
{#if $URBIT_MODE}
  <KeepAlive />
{/if}
{#if showDevPanel}
<div class="dev">
  <button on:click={printMounts}>Print Mounts</button>
  <button on:click={devStartramReminder}>Remind Startram</button>
  <button on:click={devBackupTlon}>Backup Tlon</button>
  <button on:click={()=>devStartramReminderToggle(true)}>Reminded</button>
  <button on:click={()=>devStartramReminderToggle(false)}>Have not Reminded</button>
</div>
{/if}
<style>
  .dev {
    width: 100vw;
    position: fixed;
    bottom: 0;
    left: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 16px;
  }
  .dev > button {
    padding: 6px;
    border: solid 1px black;
    border-radius: 8px;
    cursor: pointer;
  }
  .dev > button:hover {
    background: black;
    color: white;
  }
</style>
