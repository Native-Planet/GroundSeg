<script>
  import { onMount, afterUpdate } from 'svelte'
	import { updateState, api, system, noconn } from '$lib/api'
  import { page } from '$app/stores'
  import Fa from 'svelte-fa'
  import { faSatelliteDish } from '@fortawesome/free-solid-svg-icons'

  let hide = true
  let blur = false
  let data = {anchor: {wgReg:false, wgRunning: false}}

  afterUpdate(()=> {
    hide = ($page.routeId == 'login')
    blur = ($page.routeId == 'startram')
  })

	// updateState loop
  const update = () => {
    if (!$noconn) {
      fetch($api + '/anchor', {credentials: "include"})
      .then(raw => raw.json())
      .then(res => data = res)
      .catch(err => {
        if ((typeof err) == 'object') {
          updateState({status:'noconn'})
        }
      })
    }
    setTimeout(update, 10000)
	}

	// Start the update loop
	onMount(()=> {
    api.set("http://" + $page.url.hostname + ":27016")
		update()
	})

</script>

{#if !hide}
  <a href='/startram' class:hide={hide} class:blur={blur}>
    <div 
      class="img" 
      class:connected={data.anchor.wgReg && data.anchor.wgRunning}
      class:not-connected={data.anchor.wgReg && !data.anchor.wgRunning}
    >
      <Fa icon={faSatelliteDish} size="1.2x" />
    </div>
  </a>
{/if}

<style>
  .blur {
    opacity: .2;
    pointer-events: none;
    filter: blur(2px);
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
