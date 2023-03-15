<script>
	import { api, webuiVersion } from '$lib/api'
  import { scale } from 'svelte/transition'
  import PrimaryButton from '$lib/PrimaryButton.svelte'

  export let gsVersion, uiBranch, updateMode

	const toggleUpdate = () => {
    let module = 'watchtower'
	  fetch($api + '/system?module=' + module, {
			method: 'POST',
      credentials: 'include',
			headers: {'Content-Type': 'application/json'},
			body: JSON.stringify({'action':'toggle'})
	  })
      .then(d => d.json())
      .then(res => console.log(res))
  }

  const restartGroundSeg = () => {
    let module = 'binary'
	  fetch($api + '/system?module=' + module, {
			method: 'POST',
      credentials: 'include',
			headers: {'Content-Type': 'application/json'},
			body: JSON.stringify({'action':'restart'})
	  })
      .then(d => d.json())
      .then(res => console.log(res))
  }

</script>

<div class="sys">
  <div class="sys-title">System Information</div>

    <!-- groundseg_api version -->
    <div class="hw-version">
      <div class="word">Api Version</div>
      <span>{gsVersion}</span>
    </div>

    <!-- groundseg_webui version -->
    <div class="hw-version">
      <div class="word">Web UI Version</div>
      <span>{webuiVersion}{uiBranch}</span>
    </div>

    <!-- App update modes -->
    <div class="hw-version">
      <div class="word">Auto Update</div>
      <div on:click={toggleUpdate} class="switch-wrapper">
        <div class="switch {updateMode == 'auto' ? "on" : "off"}"></div>
      </div>
    </div>

    <!-- Restart groundseg -->
    <div class="hw-version">
      <div class="word">Restart GroundSeg</div>
      <PrimaryButton 
        on:click={restartGroundSeg}
        standard="Restart"
        background="black"
        />
    </div>
</div>

<style>
  .sys {
    display: flex;
    flex-direction: column;
    background: #0000001d;
    padding: 20px 30px;
    border-radius: 8px;
    font-size: 18px;
    gap: 12px;
  }
  .sys-title {
    font-size: 18px;
    padding-bottom: 8px;
  }
  .hw-version {
    display: flex;
    font-size: 14px;
    align-items: center;
  }
  .word { flex: 1; }
  .switch-wrapper {
    border-radius: 8px;
    width: 32px;
    height: 12px;
    background: #ffffff4d;
    padding: 2px;
    cursor: pointer;
  }
  .switch {
    height: 100%;
    width: 19px;
    border-radius: 6px;
    margin-top: auto;
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
