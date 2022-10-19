<script>
	import { onMount } from 'svelte'
  import { blur } from 'svelte/transition'

	let inView = false

	onMount(()=> {
		inView = !inView
	})
</script>

{#if inView}
	<div class="boot" transition:blur={{duration: 600, amount: 10}}>
		<a class="existing"	href="/boot/existing">
    	Boot an existing Urbit ID
	  </a>
		<a class="new" href="/boot/new">
    	Boot a new Urbit ID
	  </a>
	</div>
{/if}

<style>
  .boot {
    display: flex;
    flex-direction: column;
    gap: 8px;
    margin: 0 20px 20px 20px;
  }

  a {
    padding: 12px;
    color: inherit;
    text-align: center;
    font-weight: 500;
    border-radius: 8px;
  }

  .existing {
    background: #000;
  }

  .new {
    background: #8599ad8e;
  }

</style>
