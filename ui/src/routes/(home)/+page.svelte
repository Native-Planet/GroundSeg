<script>
	import { onMount, onDestroy } from 'svelte'
  import { page } from '$app/stores'
  import { structure } from '$lib/stores/websocket'
  import { scale } from 'svelte/transition'

  import Logo from '$lib/Logo.svelte'
	import Card from '$lib/Card.svelte'
	import PierList from '$lib/PierList.svelte'
	import BootButtons from '$lib/BootButtons.svelte'

	let inView = false
	onMount(()=> inView = true)
	onDestroy(()=> inView = false)

  $: urbits = ($structure?.urbits) || {}
  $: listUrbs = (Object.entries(urbits)) || []

</script>

{#if inView}
  <Card width="520px" padding={false} home={true}>
		<div style="margin: 20px 0 0 20px;">
  		<Logo />
		</div>
    <div class="wrapper">
      {#if listUrbs.length == 0}
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
        {#each listUrbs as u}
          <PierList name={u[0]} u={u[1]}/>
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
