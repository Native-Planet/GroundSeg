<script>
	import { onMount, onDestroy } from 'svelte'
  import { scale } from 'svelte/transition'
  import { page } from '$app/stores'

  import Fa from 'svelte-fa'
  import { faCheck } from '@fortawesome/free-solid-svg-icons'

	import { updateState, api, system, noconn } from '$lib/api'
  import Logo from '$lib/Logo.svelte'
	import Card from '$lib/Card.svelte'

  import AnchorHeader from '$lib/AnchorHeader.svelte'
  import AnchorRegisterKey from '$lib/AnchorRegisterKey.svelte'
  import AnchorAdvanced from '$lib/AnchorAdvanced.svelte'

	// load data into store
	export let data
  updateState(data)

	// init
	let inView = false

	// updateState loop
  const update = () => {
    if (($page.routeId == 'startram') && !$noconn) {
      fetch($api + '/anchor', {credentials: "include"})
			.then(raw => raw.json())
      .then(res => data = res)
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
		update()
    inView = true
	})

  onDestroy(()=> inView = false)
	
</script>

{#if inView}
  <Card width="460px">

    <!-- Header -->
    <AnchorHeader wgReg={data.anchor.wgReg} wgRunning={data.anchor.wgRunning}>
      <Logo t='StarTram'/>
    </AnchorHeader>

    {#if data.anchor.lease != null}
      <div class="lease" transition:scale={{duration:120, delay: 200}}>
        <span>Your subscription expires on {data.anchor.lease.slice(5,-12)}</span>
        {#if data.anchor.ongoing}
          <span class="autorenew">
            <Fa icon={faCheck} size="1x" />
            auto-renew
          </span>
        {/if}
      </div>
    {/if}

    <!-- Register Key -->
    <AnchorRegisterKey wgReg={data.anchor.wgReg} />

    <!-- Advanced Options -->
    <AnchorAdvanced wgReg={data.anchor.wgReg} />
  </Card>
{/if}

<style>
  .lease {
    padding-top: 20px;
    font-size: 12px;
  }
  .autorenew {
    margin-left: 4px;
    background: green;
    padding: 2px 8px;
    border-radius: 8px;
  }
</style>
