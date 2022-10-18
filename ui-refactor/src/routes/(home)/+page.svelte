<script>
	import { onMount, onDestroy } from 'svelte'

	import { updateState, homepageQuery, piers, api } from '$lib/api'
  import Logo from '$lib/Logo.svelte'
	import Card from '$lib/Card.svelte'
	import PierList from '$lib/PierList.svelte'
  import SettingsButton from '$lib/SettingsButton.svelte'
	import BootButtons from '$lib/BootButtons.svelte'

	// load data into store
	export let data
	updateState(data)

	// init
	let inView = false

	// updateState loop
  const update = () => {
    if (inView) {
			const query = { "data": homepageQuery()}
			let d = fetch($api, {
				method: 'POST',
				headers: {'Content-Type': 'application/json'},
				body: JSON.stringify(query)
			})
			.then(raw => raw.json())
    	.then(res => updateState(res))

			setTimeout(update, 3000)
		}
	}

	// Start the update loop
	onMount(()=> {
		inView = !inView
		update()
	})

	// end the update loop
	onDestroy(()=> inView = !inView)
	
</script>

<SettingsButton />
{#if inView}
	<Card width="460px" padding={false}>
		<div style="margin: 20px 0 0 20px;">
  		<Logo />
		</div>
		<PierList />
		<BootButtons />
	</Card>
{/if}
