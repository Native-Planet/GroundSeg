<script>
	import { onMount, onDestroy } from 'svelte'
  import { page } from '$app/stores'
	import { updateState, api, noconn } from '$lib/api'

  import Logo from '$lib/Logo.svelte'
	import Card from '$lib/Card.svelte'
	import PierList from '$lib/PierList.svelte'
	import BootButtons from '$lib/BootButtons.svelte'

	// load data into store
	export let data
	updateState(data)

	// init
	let inView = false

	// updateState loop
  const update = () => {
    if (inView && !$noconn) {
      fetch($api + '/urbits', {credentials:"include"})
			.then(raw => raw.json())
    	.then(res => updateState(res))
      .catch(err => {
        console.log(err)
        if ((typeof err) == 'object') {
          updateState({status:'noconn'})
        }
      })
			setTimeout(update, 3000)
	}}

	// Start the update loop
	onMount(()=> {
    api.set("http://" + $page.url.hostname + ":27016")
    if (data['status'] == 404) {
      window.location.href = "/login"
    }
		inView = true
		update()
	})

	// end the update loop
	onDestroy(()=> inView = false)

</script>

{#if inView}
  <Card width="520px" padding={false} home={true}>
		<div style="margin: 20px 0 0 20px;">
  		<Logo />
		</div>
		<PierList />
		<BootButtons />
	</Card>
{/if}
