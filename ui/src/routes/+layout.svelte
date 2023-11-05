<script>
  // Svelte
  import { onMount } from 'svelte'
  import { get } from 'svelte/store'
  import { page } from '$app/stores'
  import { goto } from '$app/navigation';

  // Websocket
  import { firstLoad, isC2CMode, wsPort, connect, structure, connected } from '$lib/stores/websocket'
  import { wide } from '$lib/stores/display'

  import ApiSpinner from './ApiSpinner.svelte'
  import FirstLoad from './FirstLoad.svelte'

  // Style
  import "../theme.css"

  onMount(()=> {
    const hostname = $page.url.hostname
    connect("ws://" + hostname + ":" + $wsPort + "/ws")
    redirector()
  })

  $: authLevel = ($structure?.auth_level) || "unauthorized"
  $: stage = ($structure?.stage) || null
  $: pageRouteID = $page.route.id

  let count = 0
  const redirector = () => {
    if ($connected) {
      if ($isC2CMode) {
        if (pageRouteID !== "/captive") {
          goto("/captive")
        }
      } else {
        const auth = (authLevel === "authorized")
        if (auth) {
          if ((pageRouteID === "/login") || ($page.route.id.includes("setup"))) {
            goto("/")
          }
        } else {
          if (authLevel === "unauthorized") {
            if (pageRouteID !== "/login") {
              if (count > 2) {
                count = 0
                goto("/login")
              } else {
                count += 1 
              }
            }
          }
          if (authLevel === "setup") {
            if (count > 2) {
              count = 0
              if (stage) {
                goto("/setup/" + stage)
              }
            } else {
              count += 1 
            }
          }
        }
      }
    }
    setTimeout(redirector,500)
  }

	const vert = (h,w) => {
	  let r = h / w
    let d = false
		if ( r > 1) { d = true }
		wide.set(!d)	
	}

</script>

<!--svelte:window bind:innerWidth bind:innerHeight /-->
{#if $firstLoad}
  <FirstLoad />
{:else}
  <slot/>
  <ApiSpinner />
{/if}
