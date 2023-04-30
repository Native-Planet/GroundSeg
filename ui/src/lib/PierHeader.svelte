<script>
	import { api } from '$lib/api'
	import { scale } from 'svelte/transition'

  import Fa from 'svelte-fa'
  import { faHammer } from '@fortawesome/free-solid-svg-icons'

	export let running, name

  let loading = false

	// toggle pier on or off
	const togglePier = () => {
      loading = true
			fetch($api + '/urbit?urbit_id=' + name, {
			method: 'POST',
        credentials: "include",
			headers: {'Content-Type': 'application/json'},
			body: JSON.stringify({'app':'pier','data':'toggle'})
	})
		.then(raw => raw.json())	
    .then(res => { 
      if (res == 200) {
        loading = false
      }})
		.catch(err => console.log(err))
	}

</script>

<div class="wrapper">
	<div class="slot"><slot/></div>
  <div 
    in:scale={{duration:100,delay:300, amount:10}}
    on:click={togglePier}
    class:loading={loading}
    class="switch-wrapper">
		<div class="switch {running ? "on" : "off"}"></div>
	</div>
</div>

<style>
	.wrapper {display: flex;}
	.slot {flex:1}
  .loading {
    pointer-events: none;
    opacity: .6;
  }
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
