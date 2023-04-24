<script>
	import { onMount, onDestroy } from 'svelte'
  import { scale } from 'svelte/transition'
  import { page } from '$app/stores'

  import Fa from 'svelte-fa'
  import { faCheck } from '@fortawesome/free-solid-svg-icons'

	import { updateState, api, system, noconn, startram } from '$lib/api'
  import Logo from '$lib/Logo.svelte'
	import Card from '$lib/Card.svelte'

  import AnchorHeader from '$lib/AnchorHeader.svelte'
  import AnchorInformation from '$lib/AnchorInformation.svelte'
  import AnchorRegisterKey from '$lib/AnchorRegisterKey.svelte'
  import AnchorAdvanced from '$lib/AnchorAdvanced.svelte'

	// load data into store
	export let data
  updateState(data)

	// init
	let inView = false

	// updateState loop
  const update = () => {
    if (($page.route.id == '/startram') && !$noconn) {
      fetch($api + '/anchor', {credentials: "include"})
			.then(raw => raw.json())
      .then(res => updateState(res))
      .catch(err => {
        console.log(err)
        if ((typeof err) == 'object') {
          updateState({status:'noconn'})
        }
      })

			setTimeout(update, 1000)
	}}

	// Start the update loop
	onMount(()=> {
    api.set("http://" + $page.url.hostname + ":27016")
    if (data['status'] == 404) {
      window.location.href = "/login"
    }

    if (data['status'] == 'setup') {
      window.location.href = "/setup"
    }
		update()
    inView = true
    tempGetRegions()
	})

  onDestroy(()=> inView = false)

  const tempGetRegions = () => {
    fetch($api + "/get-regions",{credentials: "include"})
      .then(r => r.json())
      .then(x => {
        if (x==200) {
          console.log("regions requested")
        } else {
          setTimeout(tempGetRegions, 3000)
        }
      })
  }
	
</script>

{#if inView}
  <Card width="460px">

    <!-- Header -->
    <AnchorHeader wgReg={$startram.wgReg} wgRunning={$startram.wgRunning}>
      <Logo t='StarTram'/>
    </AnchorHeader>

    {#if $startram.wgReg}
      <AnchorInformation
        region={$startram.region}
        regions={$startram.regions}
        ongoing={$startram.ongoing}
        lease={$startram.lease}
      />
    {/if}

    <!-- Register Key -->
    <AnchorRegisterKey
      wgReg={$startram.wgReg}
      region={$startram.region}
      regions={$startram.regions}
    />

    <div class="sign-up">
      <a href="https://www.nativeplanet.io/startram" target="_blank">
        Need a startram registration key? Get one here!
      </a>
    </div>

    <!-- Advanced Options -->
    <AnchorAdvanced wgReg={$startram.wgReg} wgRunning={$startram.wgRunning} />
  </Card>
{/if}

<style>
  .lease {
    padding-top: 20px;
    font-size: 12px;
  }
  .sign-up {
    margin-top: 12px;
    margin-left: 2px;
  }
  a {
    color: inherit;
    font-size: 12px;
    text-decoration: underline;
    cursor: pointer;
  }
</style>
