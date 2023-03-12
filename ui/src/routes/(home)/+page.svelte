<script>
	import { onMount, onDestroy } from 'svelte'
  import { page } from '$app/stores'
  import { scale } from 'svelte/transition'

	import { urbits, updateState, api, noconn } from '$lib/api'
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
    console.log(data)
    api.set("http://" + $page.url.hostname + ":27016")
    if (data['status'] == 404) {
      window.location.href = "/login"
    }

    if (data['status'] == 'setup') {
      window.location.href = "/setup"
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
    <div class="wrapper">
      {#if $urbits.length == 0}
        <div class="welcome" in:scale={{duration:120, delay: 300}}>
          Welcome to GroundSeg.
        </div>
        <div class="welcome" in:scale={{duration:120, delay: 300}}>
          From here you can boot and manage multiple Urbit IDs.
        </div>
        <div class="welcome" in:scale={{duration:120, delay: 300}}>
          Select one of the options below to get started.
        </div>
      {:else} 
        {#each $urbits as u, i}
          <PierList {u} />
        {/each}
      {/if}
    </div>
		<BootButtons />
	</Card>
{/if}
<style>
  .welcome {
    padding: 0 24px 0 24px;
    line-height: 24px;
    font-size: 14px;
  }

  .wrapper {
    margin-bottom: 28px;
    margin-top: 8px;
    display: flex;
    flex-direction: column;
    max-height: 264px;
    overflow: auto;
    -ms-overflow-style: none;
    scrollbar-width: none;
  }

  .wrapper::-webkit-scrollbar {
    display: none;
  }
</style>
