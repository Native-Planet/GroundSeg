<script>
	import { api } from '$lib/api'
	import { scale } from 'svelte/transition'
  //import { send, socket, structure } from '$lib/stores/websocket.js'
  import { structure, starTramToggle } from '$lib/stores/websocket'

  $: startram = ($structure.system?.startram) || null
  $: register = (startram?.register) || "no"
  $: container = (startram?.container) || "stopped"

	// toggle anchor on or off
	const toggleAnchor = () => {
    starTramToggle(container)
  }

</script>
<div class="wrapper">
	<div class="slot"><slot/></div>
  {#if register == "yes"}
    <div 
      in:scale={{duration:100,delay:300, amount:10}}
      on:click={toggleAnchor}
      class="switch-wrapper">
      <div class="switch {container == "running" ? "on" : "off"}" />
	  </div>
  {/if}
</div>

<style>
  .loading {
    opacity: .4;
    pointer-events: none;
  }
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
