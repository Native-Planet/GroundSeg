<script>
	import EyeButton from '$lib/EyeButton.svelte'
  import Clipboard from 'clipboard'

	export let code

	let view = false, clicked = false

	const toggleView = e => view = e.detail

  let copy = new Clipboard('#code');
 	copy.on("success", ()=> {
    clicked = true; setTimeout(()=> clicked = false, 1000)})
</script>

    <div class="pier-info">
      <div class="pier-title">Access Key</div>
      <div class="pier-cred-wrapper">
        <div on:click={copy} id="code" data-clipboard-text={code} class="pier-cred">
					{
						clicked ? "copied!" : view
						? code : "click to copy"
          }
        </div>
				<EyeButton on:click={toggleView} {view} />
      </div>
    </div>


