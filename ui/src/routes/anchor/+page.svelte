<script>
	import { onMount, onDestroy } from 'svelte'
  import { scale } from 'svelte/transition'
  import { page } from '$app/stores'

	import { updateState, api, system } from '$lib/api'
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
    if ($page.routeId == 'anchor') {
			fetch($api + '/anchor')
			.then(raw => raw.json())
    	.then(res => data = res)
			.catch(err => console.log(err))

			setTimeout(update, 1000)
	}}

	// Start the update loop
	onMount(()=> {
    inView = true
		update()
	})

  onDestroy(()=> inView = false)
	
</script>

{#if inView}
  <Card width="460px">
    <!-- Header -->
    <AnchorHeader wgReg={data.anchor.wgReg} wgRunning={data.anchor.wgRunning}>
      <Logo t='Anchor'/>
    </AnchorHeader>

    {#if data.anchor.lease != null}
      <div class="lease" transition:scale={{duration:120, delay: 200}}>
        Your subscription expires on {data.anchor.lease.slice(5,-12)}
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
</style>
