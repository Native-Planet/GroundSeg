<script>
  import Fa from 'svelte-fa'
  import { faCircleQuestion } from '@fortawesome/free-regular-svg-icons'
  import { api } from '$lib/api'
  import PrimaryButton from '$lib/PrimaryButton.svelte'

  export let name, loomSize

  let showLoom = false
  let modLoomStatus = "standard"
  let curLoomSize = loomSize

  const toggleLoom = () => showLoom = !showLoom

  const modLoomSize = () => {
    modLoomStatus = "loading"
		fetch($api + '/urbit?urbit_id=' + name, {
  		method: 'POST',
      credentials: "include",
	  	headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({'app':'pier','data':'loom','size':curLoomSize}),
    })
      .then(r => r.json())
      .then(d => {
        if (d == 200) {
          modLoomStatus = "success"
        } else {
          modLoomStatus = "failure"
        }
        setTimeout(()=> modLoomStatus = "standard", 3000)
      })
  }
</script>

<div class="bg">
  <div class="option-title">
    Loom Size
    <button class="question-mark" on:click={toggleLoom} >
      <Fa icon={faCircleQuestion} size="1.2x" />
    </button>
  </div>
  {#if showLoom}
    <div class="loom-info">
      Loom settings set the amount of memory your ship is allocated in megabytes. 
      Do not go below 2048MB if you do not know what you are doing!
    </div>
  {/if}

  <div class="loom">
    <input type="range" min="28" max="32" step="1" class="range" bind:value={curLoomSize}>
    <div class="labels">
      <div class="label">256</div>
      <div class="label">512</div>
      <div class="label">1024</div>
      <div class="label">2048</div>
      <div class="label">4096</div>
    </div>

    <PrimaryButton
      on:click={modLoomSize}
      noMargin={true}
      status={(loomSize == curLoomSize) && (modLoomStatus == "standard") ? "disabled" : modLoomStatus}
      standard="Modify Loom Size"
      success="Loom size modified!"
      failure="Something went wrong, try again"
      loading="Modifying..."
    />
  </div>
</div>

<style>
  .bg {
    background: #0000001d;
    padding: 20px 0 20px 0;
    border-radius: 12px;
  }
  .option-title {
    font-size: 14px;
    color: inherit;
    margin-bottom: 4px;
  }
  .question-mark {
    color: inherit;
    cursor: pointer;
  }
  .loom {
    display: flex;
    align-items: center;
    flex-direction: column;
    gap: 4px;
  }
  .loom-info {
    font-size: 11px;
  }
  .range {
    -webkit-appearance: none;
    width: 180px;
    height: 11px;
    background: #ffffff4d;
    border-radius: 16px;
  }
  .range::-webkit-slider-thumb {
    -webkit-appearance: none;
    appearance: none;
    width: calc(180px / 5);
    height: 16px;
    background: var(--action-color);
    cursor: pointer;
    border-radius: 16px;
  }
  .range::-moz-range-thumb {
    width: calc(180px / 5);
    height: 16px;
    background: var(--action-color);
    cursor: pointer;
    border-radius: 16px;
  }
  .labels {
    display: flex;
    justify-content: space-between;
    width: 180px;
    padding-bottom: 6px;
  }
  .label {
    width: calc(180px / 5);
    font-size: 11px;
  }
</style>
