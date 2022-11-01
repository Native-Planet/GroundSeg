<script>
	import { api } from '$lib/api'
	import { scale } from 'svelte/transition'

	export let running, name

	// toggle pier on or off
	const togglePier = () => {
			fetch($api + '/urbit?urbit_id=' + name, {
			method: 'POST',
			headers: {'Content-Type': 'application/json'},
			body: JSON.stringify({'app':'pier','data':'toggle'})
	})
		.then(raw => raw.json())	
		.then(res => { console.log(res)})
		.catch(err => console.log(err))
	}

</script>

<div class="wrapper">
	<div class="slot"><slot/></div>
	<div in:scale={{duration:100,delay:300, amount:10}} on:click={togglePier} class="switch-wrapper">
		<div class="switch {running ? "on" : "off"}"></div>
	</div>
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
