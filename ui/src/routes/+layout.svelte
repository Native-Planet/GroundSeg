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
  import DevPanel from '$lib/DevPanel.svelte'

  // Style
  import "../theme.css"

  const isUrbitMode = process.env.GS_URBIT_MODE;
  const showDevPanel = process.env.GS_DEV_PANEL;
  const customHostname = process.env.GS_CUSTOM_HOSTNAME;

  onMount(()=> {
    URBIT_MODE.set(isUrbitMode)
    DEV_PANEL.set(showDevPanel)
    const hostname = $page.url.hostname
    if ($URBIT_MODE) {
      subscribe(window.ship)
    } else {
      if (customHostname) {
        connect("ws://" + customHostname + ":" + $wsPort + "/ws") // if GS_CUSTOM_HOSTNAME is set
      } else {
        connect("ws://" + hostname + ":" + $wsPort + "/ws")
      }
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
  <DevPanel />
{/if}