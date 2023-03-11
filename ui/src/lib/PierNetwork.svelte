<script>
	import { api } from '$lib/api'

	export let name, remote, wgReg, wgRunning

	let isSwitching = false

	// toggle network
  const toggleNetwork = () => { 
		isSwitching = true
		fetch($api + '/urbit?urbit_id=' + name, {
		method: 'POST',
        credentials: "include",
		headers: {'Content-Type': 'application/json'},
		body: JSON.stringify({'app':'wireguard','data':'toggle'})
		})
		.then(raw => raw.json())	
		.then(res => { console.log(res); isSwitching = false})
		.catch(err => console.log(err))
	}

</script>
{#if wgReg && wgRunning}
  <div class="pier-info" class:switching={isSwitching}>
    <div class="pier-title">Connectivity</div>
    <div class="access-options" on:click={toggleNetwork}>
			<button class="option" class:access-active={remote == false} >Local</button>
      <button class="option" class:access-active={remote == true} >Remote</button>
    </div>
	</div>
{/if}

<style>
  .access-options {
    display: flex;
    width: 180px;
    border-radius: 8px;
    background: #ffffff4d;
    gap: 2px;
  }
  .option {
    color: inherit;
    font-size: 12px;
    flex: 1;
    padding: 8px 0 8px 0;
    background: none;
    border-radius: 8px;
    border: none;
    font-weight: 700;
    cursor: pointer;
  }
  .switching {
    opacity: .6;
    pointer-events: none;
  }
  .access-active {
    background: #008eff;
  }
</style>
