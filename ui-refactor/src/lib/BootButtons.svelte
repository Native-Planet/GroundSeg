<script>
	import { onMount } from 'svelte'
  import { fade } from 'svelte/transition'
	import { quintOut } from 'svelte/easing'

	let inView = false

	onMount(()=> {
		inView = !inView
	})
</script>

{#if inView}
	<div class="boot">
		<a
			class="existing"
			href="/boot/existing"
			in:fade={{delay: 400, duration: 300, easing: quintOut }}>
    	Boot an existing Urbit ID
	  </a>
		<a
			class="new"
		 	href="/boot/new"
			in:fade={{delay: 460, duration: 300, easing: quintOut }}>
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
