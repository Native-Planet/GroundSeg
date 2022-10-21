<script>
  import ExtUrl from '$lib/ExtUrl.svelte'
	import EyeButton from '$lib/EyeButton.svelte'
  import Clipboard from 'clipboard'

	export let urbitUrl

	let view = false, clicked = false

	const toggleView = e => view = e.detail

  let copy = new Clipboard('#urbitUrl');
 	copy.on("success", ()=> {
  clicked = true; setTimeout(()=> clickedExt = false, 1000)})
</script>

<div class="pier-info">
  <div class="pier-title">Urbit Link</div>
  <div class="pier-cred-wrapper">
    <div on:click={copy} id="urbitUrl" data-clipboard-text={urbitUrl} class="pier-cred">
			{
				clicked ? "copied!" : view
				? urbitUrl : "click to copy"
      }
    </div>
		<ExtUrl link={urbitUrl} />
		<EyeButton on:click={toggleView} {view} />
  </div>
</div>
