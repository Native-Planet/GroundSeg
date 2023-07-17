<script>
  import { fade } from 'svelte/transition'
  import { fadeIn, fadeOut, isPortrait } from '$lib/api'

  export let width
  export let maxHeight = "80vh"
  export let padding = true
  export let home = false
  export let bgColor = "#6d6d6d33"
  export let devMode = false

  if (devMode) {
    bgColor = "#FFF7C422"
  }

	let innerWidth = 0
  let innerHeight = 0

	const vert = (h,w) => {
	  let r = h / w
		if ( r > 1) { return true }
		return false	
	}
</script>

<div in:fade={fadeIn}
		 out:fade={fadeOut} 
		 class="card" 
	 	 style="padding:{padding ? '20px' : '0'};
      max-height:{maxHeight}; width: {
      $isPortrait
      ? home ? '100vw' : 'calc(100vw - 40px)'
			  : width
        };
      background: {bgColor};">
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
    
    border-radius: 20px;

    backdrop-filter: blur(60px);
    -moz-backdrop-filter: blur(60px);
    -o-backdrop-filter: blur(60px);
    -webkit-backdrop-filter: blur(60px);

    -ms-overflow-style: none;
		scrollbar-width: none;

    overflow: scroll;
	}
</style>
