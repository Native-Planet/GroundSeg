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
  console.log(data)

  let inViewSetup = false
  let setupPage = 0

  onMount(()=> {
    api.set("http://" + $page.url.hostname + ":27016")
    inViewSetup = true
  })

</script>

{#if inViewSetup}
  <Card width="480px">
    <Logo t='GroundSeg Setup'/>

    {#if setupPage == 0}
      <SetupPassword on:nextPage={()=> setupPage = 1} />
    {/if}

    {#if setupPage == 1}
      <SetupAnchor on:prevPage={()=> setupPage = 0}/>
    {/if}

    <div class="page-num">{setupPage + 1} of 2</div>
  </Card>
{/if}

<style>
  .page-num {
    text-align: left;
    font-size: 10px;
    margin-top: 12px;
  }
</style>
