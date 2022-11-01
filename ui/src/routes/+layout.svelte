<script>
  import { afterUpdate } from 'svelte'

  import { isPortrait } from '$lib/api'
  import SettingsButton from '$lib/SettingsButton.svelte'
  import AnchorButton from '$lib/AnchorButton.svelte'

	let innerWidth = 0
  let innerHeight = 0

	const vert = (h,w) => {
	  let r = h / w
    let d = false
		if ( r > 1) { d = true }
		isPortrait.set(d)	
	}

  afterUpdate(()=>vert(innerHeight, innerWidth))
</script>

<svelte:window bind:innerWidth bind:innerHeight />
<div>
<SettingsButton />
<AnchorButton />
  <slot/>
</div>

<style>
  @font-face {
    font-family: Inter;
    src: url("/Inter-SemiBold.otf");
  }
  div {
    font-family:Inter;
    background: url("/background.png") no-repeat center center fixed;
    -webkit-background-size: contain;
    -moz-background-size: contain;
    -o-background-size: contain;
    background-size: contain;
    background-color: #040404;
    height: 100vh;
    width: 100vw;
    --action-color: #008eff;
  }
</style>
