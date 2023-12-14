<script>
  // Urbit
  import { broadcast, subscribe } from '$lib/stores/urbit.js'
  // Svelte
  import { onMount } from 'svelte'
  import { page } from '$app/stores'

  // Websocket
  import { wsPort, connect } from '$lib/stores/websocket'
  import { firstLoad, URBIT_MODE } from '$lib/stores/data'
  import { wide } from '$lib/stores/display'

  import Redirector from './redirector.svelte'
  import ApiSpinner from './apispinner.svelte'
  import KeepAlive from './keepalive.svelte'
  import FirstLoad from './firstload.svelte'

  // Style
  import "../theme.css"

  const isUrbitMode = process.env.GS_URBIT_MODE;

  onMount(()=> {
    URBIT_MODE.set(isUrbitMode)
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
