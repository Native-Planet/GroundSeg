<script>
  import ExtUrl from '$lib/ExtUrl.svelte'
	import EyeButton from '$lib/EyeButton.svelte'
  import Clipboard from 'clipboard'

	export let minIOUrl, minIOReg, remote

	let view = false, clicked = false

	const toggleView = e => view = e.detail

  let copy = new Clipboard('#minIOUrl');
 	copy.on("success", ()=> {
  clicked = true; setTimeout(()=> clicked = false, 1000)})
</script>

{#if minIOReg && remote}
  <div class="pier-info">
    <div class="pier-title">MinIO Local Storage Console</div>
    <div class="pier-cred-wrapper">
      <div on:click={copy} id="minIOUrl" data-clipboard-text={minIOUrl} class="pier-cred">
		  	{
			  	clicked ? "copied!" : view
				  ? minIOUrl : "click to copy"
        }
      </div>
		  <ExtUrl link={minIOUrl} />
  		<EyeButton on:click={toggleView} {view} />
    </div>
  </div>
{/if}
