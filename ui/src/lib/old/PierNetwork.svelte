<script>
	import { api } from '$lib/api'
  import { structure, urbitsAccessToggle } from "$lib/stores/websocket" 

	export let name, remote, wgReg, wgRunning

	let isSwitching = false

	// toggle network
  const accessToggle = () => { 
    urbitsAccessToggle(name)
  }

</script>
{#if wgReg && wgRunning}
  <div class="pier-info" class:switching={isSwitching}>
    <div class="pier-title">Connectivity</div>
    <div class="access-options" on:click={accessToggle}>
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
