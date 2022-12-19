<script>
	import { onMount } from 'svelte'
	import { updateState, api } from '$lib/api'
  import { scale } from 'svelte/transition'

  import { page } from '$app/stores'

  import Card from '$lib/Card.svelte'
  import Logo from '$lib/Logo.svelte'
  import SetupAnchor from '$lib/SetupAnchor.svelte'
  import SetupPassword from '$lib/SetupPassword.svelte'

	// load data into store
	export let data
	updateState(data)

  let inViewSetup = false, setupPage = 0

  onMount(()=> {
    api.set("http://" + $page.url.hostname + ":27016")
    inViewSetup = true
  })

</script>

{#if inViewSetup}
  <Card width="480px">
    <Logo t='GroundSeg Setup'/>

    {#if setupPage == 0}
      <SetupAnchor on:nextPage={()=> setupPage = 1}/>
    {/if}

    {#if setupPage == 1}
      <SetupPassword on:prevPage={()=> setupPage = 0} />
    {/if}

  </Card>
{/if}


