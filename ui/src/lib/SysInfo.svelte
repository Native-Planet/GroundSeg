<script>
	import { api, webuiVersion } from '$lib/api'
  import { scale } from 'svelte/transition'
  import PrimaryButton from '$lib/PrimaryButton.svelte'

  export let ram, temp, disk, cpu, gsVersion, uiBranch, updateMode

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

		<!-- RAM info -->
    <div class="hw">
      <div class="word">RAM</div>
      <div class="data">{ram}%</div>
		</div>

		<!-- CPU info -->
    <div class="hw" in:scale={{duration:120}}>

      <div class="word">CPU Temperature</div>
      <div class="data">{temp} &deg C</div>
		</div>

    <!-- CPU usage -->
    <div class="hw">
      <div class="word">CPU Load</div>
      <div class="data">{cpu}%</div>
		</div>


		<!-- Hard Disk storage -->
    <div class="hw">
      <div class="word">Storage</div>
      <div class="data">
        <span>{(disk[1] / (1000 * 1000 * 1000)).toFixed(1)}</span> 
        <span>GB / </span>
        <span>
          {(disk[0] / (1000 * 1000 * 1000)).toFixed(1)} GB
        </span>
      </div>
    </div>

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
    background: #0404044d;
    padding: 40px;
    border-radius: 15px;
    font-size: 18px;
    gap: 12px;
  }
  .sys-title {
    font-size: 18px;
    padding-bottom: 8px;
  }
  .hw {
    display: flex;
    font-size: 14px;
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
