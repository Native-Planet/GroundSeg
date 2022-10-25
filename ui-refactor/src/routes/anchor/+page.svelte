<script>
	import { onMount, onDestroy } from 'svelte'

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
    if (inView) {
			fetch($api + '/system')
			.then(raw => raw.json())
    	.then(res => updateState(res))
			.catch(err => console.log(err))

			setTimeout(update, 1000)
	}}

	// Start the update loop
	onMount(()=> {
		inView = !inView
		update()
	})

	// end the update loop
	onDestroy(()=> inView = !inView)
	
</script>

{#if inView}
  <Card width="460px">

    <!-- Header -->
    <AnchorHeader wgReg={$system.wgReg} wgRunning={$system.wgRunning}>
      <Logo t='Anchor'/>
    </AnchorHeader>

    <!-- Register Key -->
    <AnchorRegisterKey wgReg={$system.wgReg} />

    <!-- Advanced Options -->
    <AnchorAdvanced wgUrl={$system.wgUrl}/>
  </Card>
{/if}
