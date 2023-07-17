<script>
  import ExtUrl from '$lib/ExtUrl.svelte'
	import EyeButton from '$lib/EyeButton.svelte'
  import Clipboard from 'clipboard'

  export let minIOUrl

	let view = false, clicked = false

	const toggleView = e => view = e.detail

  let copy = new Clipboard('#minIOUrl');
 	copy.on("success", ()=> {
  clicked = true; setTimeout(()=> clicked = false, 1000)})
</script>

<div class="title">MinIO Local Storage Console</div>
<div class="option-cred-wrapper">
  <div on:click={copy} id="minIOUrl" data-clipboard-text={minIOUrl} class="option-cred">
    {
      clicked ? "copied!" : view
      ? minIOUrl : "click to copy"
    }
  </div>
  <div class="button-wrapper">
    <EyeButton on:click={toggleView} {view} advanced={true} />
    <ExtUrl link={minIOUrl} advanced={true} />
  </div>
</div>

<style>
  .title {
    font-size: 12px;
  }
  .button-wrapper {
    display: flex;
    gap: 12px;
    align-items: center;
    justify-content: center;
    margin-bottom: 16px;
  }
</style>
