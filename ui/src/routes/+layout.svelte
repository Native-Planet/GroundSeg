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
  import { runtimeModeConfig } from '$lib/runtime/config/mode-config.js'
  import { startRuntimeSession } from '$lib/runtime/session/session-orchestrator.js'

  import Redirector from './Redirector.svelte'
  import ApiSpinner from './ApiSpinner.svelte'
  import KeepAlive from './KeepAlive.svelte'
  import FirstLoad from './FirstLoad.svelte'
  import DevPanel from '$lib/DevPanel.svelte'

  // Style
  import "../theme.css"

  onMount(()=> {
    startRuntimeSession({
      pageUrl: $page.url,
      wsPort: $wsPort,
      urbitModeEnabled: $URBIT_MODE,
      customHostname: runtimeModeConfig.customHostname,
      connectWebsocket: connect,
      subscribeUrbit: subscribe,
      ship: window.ship
    })
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
{#if $DEV_PANEL}
  <DevPanel />
{/if}
