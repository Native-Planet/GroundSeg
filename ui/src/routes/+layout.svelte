<script>
  // WebSocket Store
  import { connect } from "$lib/stores/websocket.js" 

  import { onMount, afterUpdate } from 'svelte'
  import { get } from 'svelte/store'
  import { page } from '$app/stores'
  import { power, api, isPortrait, noconn } from '$lib/api'

  import SettingsButton from '$lib/SettingsButton.svelte'
  import AnchorButton from '$lib/AnchorButton.svelte'
  import HomeButton from '$lib/HomeButton.svelte'
  //import LinuxButton from '$lib/LinuxButton.svelte'
  import BugButton from '$lib/BugButton.svelte'

  import PowerScreen from '$lib/PowerScreen.svelte'
  import NoConnection from '$lib/NoConnection.svelte'

	let innerWidth = 0
  let innerHeight = 0

	const vert = (h,w) => {
	  let r = h / w
    let d = false
		if ( r > 1) { d = true }
		isPortrait.set(d)	
	}

  const checkStatus = () => {
    if ($noconn) {
      fetch($api + "/cookies",{credentials:"include"})
        .then(() => {
          noconn.set(false)
          setTimeout(checkStatus, 15000)
        })
        .catch(err => {
          setTimeout(checkStatus, 2000)
        })
    } else {
      setTimeout(checkStatus, 15000)
    }
  }

  afterUpdate(()=> {
    vert(innerHeight, innerWidth)
    if ($page.url.pathname != '/settings') {
      power.set(null)
    }
  })

  onMount(()=> {
    api.set("http://" + $page.url.hostname + ":27016")
    checkStatus()
    connect("ws://" + $page.url.hostname + ":8000", document.cookie)
  })

</script>

<svelte:window bind:innerWidth bind:innerHeight />

<PowerScreen />

<div class="bg">
  {#if $noconn}
    <NoConnection />
  {:else}
    <div class:frozen={($page.url.pathname === "/settings") 
      && (($power === 'shutdown') || ($power === 'restart'))}>
      {#if !($page.route.id == '/setup')}
      <SettingsButton />
      <AnchorButton />
      <HomeButton />
      <!--LinuxButton /-->
      {/if}
      <slot/>
      <BugButton />
    </div>
  {/if}
</div>

<style>
  @font-face {
    font-family: Inter;
    src: url("/Inter-SemiBold.otf");
  }
  div {
    font-family:Inter;
    background: url("/background") no-repeat center center fixed;
    -webkit-background-size: auto;
    -moz-background-size: auto;
    -o-background-size: auto;
    background-size: auto;
    background-color: #040404;
    height: 100vh;
    width: 100vw;
    --action-color: #008eff;
  }
  .frozen {
    opacity: 0;
    pointer-events: none;
  }
</style>
