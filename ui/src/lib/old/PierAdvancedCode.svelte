<script>
	import EyeButton from '$lib/EyeButton.svelte'
  import Clipboard from 'clipboard'

	export let code
  export let disabled

	let view = false, clicked = false

	const toggleView = e => view = e.detail

  let copy = new Clipboard('#code');
 	copy.on("success", ()=> {
    clicked = true; setTimeout(()=> clicked = false, 1000)})
</script>

<div class="bg" class:disabled={disabled}>
  <div class="option-title">Access Key</div>
  <div class="option-cred-wrapper">
    <div on:click={copy} id="code" data-clipboard-text={code} class="option-cred">
      {
        clicked ? "copied!" : view
        ? code : "click to copy"
      }
    </div>
    <div style="height:32px;">
      <EyeButton on:click={toggleView} {view} advanced={true} />
    </div>
  </div>
</div>

<style>
  .bg {
    background: #0000001d;
    padding: 20px 0 20px 0;
    border-radius: 12px;
  }
  .disabled {
    opacity: .6;
    pointer-events: none;
    background: #FF000033;
    color: #ffffff4d;
  }
</style>
