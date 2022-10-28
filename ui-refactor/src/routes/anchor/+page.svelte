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
	let inView = true

	// updateState loop
  const update = () => {
    if (inView) {
			fetch($api + '/anchor')
			.then(raw => raw.json())
    	.then(res => data = res)
			.catch(err => console.log(err))

			setTimeout(update, 1000)
	}}

	// Start the update loop
	onMount(()=> {
		update()
	})

	// end the update loop
  onDestroy(()=> inView = false)
	
</script>

{#if inView}
  <Card width="460px">
    <!-- Header -->
    <AnchorHeader wgReg={data.anchor.wgReg} wgRunning={data.anchor.wgRunning}>
      <Logo t='Anchor'/>
    </AnchorHeader>

    <!-- Register Key -->
    <AnchorRegisterKey wgReg={data.anchor.wgReg} />

    <!-- Advanced Options -->
    <AnchorAdvanced />
  </Card>
{/if}
