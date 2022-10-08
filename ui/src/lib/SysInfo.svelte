<script>
	import { api } from '$lib/api'
  import Select from 'svelte-select'

  export let info

	const toggleUpdate = () => {
    let u = api + "/settings/update"
    const f = new FormData()
		if (info.updateMode == 'auto') {
    	f.append('updateMode', 'off')
		} else {
			f.append('updateMode', 'auto')
		}

    fetch(u, {method: 'POST',body: f})
			.then(r => r.json())
      .then(d => console.log(d))
	}

</script>

<div class="sys">
  <div class="sys-title">System Information</div>

  <!-- If data is loaded -->
  {#if info}
		
		<!-- RAM info -->
    <div class="hw">
      <div class="word">RAM</div>
      <div class="data">{info.ram}%</div>
		</div>

		<!-- CPU info -->
    <div class="hw">
      <div class="word">CPU Temperature</div>
      <div class="data">{info.temp} &deg C</div>
		</div>

		<!-- Hard Disk storage -->
    <div class="hw">
      <div class="word">Storage</div>
      <div class="data">
        <span>{(info.disk[1] / (1000 * 1000 * 1000)).toFixed(1)}</span> 
        <span>GB / </span>
        <span>
          {(info.disk[0] / (1000 * 1000 * 1000)).toFixed(1)} GB
        </span>
      </div>
    </div>

		<!-- groundseg_api version -->
    <div class="hw-version">
      <div class="word">Api Version</div>
      <span>{info.gsVersion}</span>
    </div>

		<!-- groundseg_webui version -->
    <div class="hw-version">
      <div class="word">Web UI Version</div>
      <span>Beta-1.0.0</span>
    </div>

		<!-- App update modes -->
    <div class="hw-version">
      <div class="word">Auto Update</div>
      <div on:click={toggleUpdate} class="switch-wrapper">
        <div class="switch {info.updateMode == 'auto' ? "on" : "off"}"></div>
      </div>
    </div>

	<!-- If data is not loaded -->
  {:else}
    <div class="hw">
      <div class="word">RAM</div>
      <div class="data blurred"></div>
    </div>
    <div class="hw">
      <div class="word">CPU Temperature</div>
      <div class="data blurred"></div>
    </div>
    <div class="hw">
      <div class="word">Storage</div>
      <div class="data blurred-long"></div>
    </div>
    <div class="hw">
      <div class="word">Version</div>
      <div class="data blurred"></div>
    </div>

  {/if}
</div>

<style>
  @keyframes breathe {
    0% {opacity: .6}
    50% {opacity: 0}
    100% {opacity: .6}
  }

  .sys {
    display: flex;
    flex-direction: column;
    background: #0000006d;
    width: 300px;
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
  .word {
    flex: 1;
  }

  .blurred {
    width: 60px;
    animation: breathe 2s infinite;
    background: #ffffff4d;
    filter: blur(6px);
    border-radius: 8px;
  }
  .blurred-long {
    width: 160px;
    animation: breathe 2s infinite;
    background: #ffffff4d;
    filter: blur(6px);
    border-radius: 8px;
  }
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
  .switch-wrapper-blurred {
    border-radius: 8px;
    width: 32px;
    height: 12px;
    background: #ffffff4d;
    padding: 2px;
    filter: blur(10px);
    animation: breathe 2s infinite;
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
