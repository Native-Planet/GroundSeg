<script>
	import { api } from '$lib/api'
	import { scale } from 'svelte/transition'

	export let wgRunning, wgReg

	// toggle anchor on or off
	const toggleAnchor = () => {
    let module = 'anchor'
	  fetch($api + '/system?module=' + module, {
			method: 'POST',
			headers: {'Content-Type': 'application/json'},
			body: JSON.stringify({'action':'toggle'})
	  })
      .then(d => d.json())
      .then(res => console.log(res))
  }

</script>
<div class="wrapper">
	<div class="slot"><slot/></div>
  {#if wgReg}
	  <div in:scale={{duration:100,delay:300, amount:10}} on:click={toggleAnchor} class="switch-wrapper">
      <div class="switch {wgRunning ? "on" : "off"}" />
	  </div>
  {/if}
</div>

<style>
	.wrapper {display: flex;}
	.slot {flex:1}
  .switch-wrapper {
    cursor: pointer;
    border-radius: 8px;
    width: 32px;
    height: 12px;
    background: #ffffff4d;
    padding: 2px;
    margin-bottom: 6px;
  }
  .switch {
    height: 100%;
    width: 19px;
    border-radius: 6px;
  }
  .on {
    background: #008eff;
    float: right;
  }
  .off {
    background: #000;
    float: left;
    opacity: .2;
  }
</style>
