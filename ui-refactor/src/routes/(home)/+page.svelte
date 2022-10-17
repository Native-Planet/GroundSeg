<script>
	import { onMount, onDestroy } from 'svelte'
	import { updateState, piers, api } from '$lib/api'
  import Logo from '$lib/Logo.svelte'
	import Card from '$lib/Card.svelte'
  import SettingsButton from '$lib/SettingsButton.svelte'

	// load data into store
	export let data
	//updateState(data)
	console.log(data)

	// is page in view
	let inView = true

	// updateState loop
  const update = () => {
    if (inView) {
			const query = { "data": "query { piers }"}
			let d = fetch($api, {
				method: 'POST',
				headers: {'Content-Type': 'application/json'},
				body: JSON.stringify(query)
			})
			.then(raw => raw.json())
    	.then(res => updateState(res))

			setTimeout(update, 1000)
		}
	}

	// Start the update loop
	//onMount(()=> update())

	// end the update loop
	onDestroy(()=> inView = !inView)
	
</script>

<SettingsButton />
<Card width="460px">
  <Logo />
	<a href="nallux-dozryl">Legit</a>
	<a href="tsar0s">should redirect</a>
	<a href="boot/existing">Existing</a>
	<a href="boot/new">New</a>
</Card>
