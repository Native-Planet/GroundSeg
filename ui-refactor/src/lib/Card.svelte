<script>
  import { fade } from 'svelte/transition'
  import { fadeIn, fadeOut } from '$lib/api'

	export let width, maxHeight = "80vh", padding = true

	let innerWidth = 0
  let innerHeight = 0

	const vert = (h,w) => {
	  let r = h / w
		if ( r > 1) { return true }
		return false	
	}
</script>

<svelte:window bind:innerWidth bind:innerHeight />

<div in:fade={fadeIn}
		 out:fade={fadeOut} 
		 class="card" 
	 	 style="padding:{padding ? '20px' : '0'};
		 	max-height:{maxHeight}; width: { vert(innerHeight, innerWidth)
			 ? 'calc(100vw - 40px)'
			 : width
			 };">
	<slot/>
</div>

<style>

	.card::-webkit-scrollbar {display: none;}
	.card {
    position: fixed;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    color: #fff;
    
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 20px;

    background: rgba(109, 109, 109, 0.5);
    backdrop-filter: blur(60px);
    -moz-backdrop-filter: blur(60px);
    -o-backdrop-filter: blur(60px);
    -webkit-backdrop-filter: blur(60px);

    -ms-overflow-style: none;
		scrollbar-width: none;

    overflow: scroll;
	}

</style>
