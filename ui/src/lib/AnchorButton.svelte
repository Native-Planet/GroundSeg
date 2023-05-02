<script>
  import { onMount, afterUpdate } from 'svelte'
  import { page } from '$app/stores'

  import { socketInfo } from '$lib/stores/websocket.js'

  import Fa from 'svelte-fa'
  import { faSatelliteDish } from '@fortawesome/free-solid-svg-icons'

  $: startram = ($socketInfo.system?.startram) || {}
  $: register = (startram?.register) || "no"
  $: container = (startram?.container) || "stopped"

  let hide = true
  let blur = false

  afterUpdate(()=> {
    hide = ($page.route.id == '/login')
    blur = ($page.route.id == '/startram')
  })

</script>

{#if !hide}
  <a href='/startram' class:hide={hide} class:blur={blur}>
    <div 
      class="img" 
      class:connected={(register == "yes") && (container == "running")}
      class:not-connected={(register == "yes") && !(container == "running")}
    >
      <Fa icon={faSatelliteDish} size="1.2x" />
    </div>
  </a>
{/if}

<style>
  .blur {
    opacity: .3;
    pointer-events: none;
  }
  .hide {
    opacity: 0;
    pointer-events: none;
  }
	a {
		position:absolute;
		left: 44px;
    top: 3px;
	}
  .img {height: 24px; margin: 20px; color: white}
  .connected {
    color: lime;
  }
  .not-connected {
    color: red;
  }
</style>
