<script>
	import { onMount } from 'svelte'
  import { scale } from 'svelte/transition'

	import { piers } from '$lib/api'
	import Sigil from '$lib/Sigil.svelte'

  import Fa from 'svelte-fa'
  import { faGear } from '@fortawesome/free-solid-svg-icons'

	let inView = false

	onMount(()=> {
		inView = !inView
	})

</script>
	<div class="wrapper">
 		{#each $piers as p, i}
			{#if inView}
		 		<div class="pier" in:scale={{duration:160, delay: 360 + (50 * i)}}>
			   	<Sigil patp={p.name} size="60px" rad="8px" />
					<a class="info"
    	  		href={p.running ? p.urbitUrl : ""}
		      	target={p.running ? "_blank" : ""}>
	  		    <div class="patp">{p.name}</div>
  	    		<div class="status">{p.running ? 'Running' : 'Stopped'}</div>
		    	</a>
	  		  <a href={p.name}>
  	    		<Fa icon={faGear} size="1.2x" />
  	    	</a>
			  </div>
			{/if}
		 {/each}
	</div>
<style>
	a { color: inherit; }

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

  .pier {
    padding: 6px 20px 6px 20px;
    display: flex;
    align-items: center;
    flex-wrap: wrap;
  }
  .pier:hover {
    background: #00000080;
  }
  .info {
    display: flex;
    flex-direction: column;
    padding-left: 12px;
    flex: auto;
  }
  .patp {
    font-weight: 600;
    color: inherit;
  }
  .status {
    opacity: .8;
    font-weight: 400;
    font-size: .8em;
  }
</style>
