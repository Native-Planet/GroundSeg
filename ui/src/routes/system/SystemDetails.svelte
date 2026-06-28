<script>
  import { updateLinux, restartGroundSeg, setSwap, checkUpdates } from '$lib/stores/websocket'
  import { connected, structure } from '$lib/stores/data'

  import { afterUpdate } from 'svelte'

  import Swap from './SysDetails/Swap.svelte'
  import Ram from './SysDetails/Ram.svelte'
  import Storage from './SysDetails/Storage.svelte'
  import CPULoad from './SysDetails/CPULoad.svelte'
  import CPUTemp from './SysDetails/CPUTemp.svelte'
  import SMARTCheck from './SysDetails/SMARTCheck.svelte'

  $: usage = ($structure?.system?.info?.usage) || {}
  $: swap = (usage?.swap) || 0
  $: ram = (usage?.ram) || [0,0]
  $: disk = (usage?.disk) || {};
  $: cpu = (usage?.cpu) || 0
  $: cpuTemp = (usage?.cpu_temp) || 0
  $: smart = ($structure?.system?.info?.smart) || {}
  $: updateStatus = ($structure?.system?.transition?.checkUpdates) || ""
  $: checkingUpdates = updateStatus.length > 0 && !["success", "up-to-date", "error"].includes(updateStatus)
  $: updateButtonText = updateStatusText(updateStatus)

  let restarting = false
  let dead = false
  let success = false

  const handleGroundSegRestart = () => {
    restarting = true
    restartGroundSeg()
  }

  function updateStatusText(status) {
    if (!status) return "Check for Updates"
    if (status == "queued") return "Queued"
    if (status == "checking") return "Checking..."
    if (status == "up-to-date") return "Up to Date"
    if (status == "success") return "Updates Checked"
    if (status == "error") return "Check Failed"
    if (status.startsWith("updating ")) {
      return `Updating ${status.replace("updating ", "")}`
    }
    return status
  }

  afterUpdate(()=> {
    if (restarting) {
      if (!$connected) {
        dead = true
      } else {
        if (dead) {
        restarting = false
        success = true
          setTimeout(()=>{
            success = false
            dead = false
          }
          , 3000)
        }
      }
    }
  })
</script>

<div class="container">
  <div class="sys-title">
    <span>SYSTEM DETAILS</span>
    <div class="title-actions">
      <button on:click={checkUpdates} class="restart-groundseg secondary" disabled={checkingUpdates}>
        {updateButtonText}
      </button>
      <button on:click={handleGroundSegRestart} class="restart-groundseg" disabled={restarting}>
        {#if success}
          GroundSeg Restarted!
        {:else}
          {restarting ? "GroundSeg Restarting..." : "Restart GroundSeg"}
        {/if}
      </button>
    </div>
  </div>
  <div class="item-wrapper">
    <Swap {swap}/>
  </div>
  <div class="item-wrapper">
    <Ram {ram} />
  </div>
  <div class="item-wrapper">
    <CPULoad {cpu} />
  </div>
  <div class="item-wrapper">
    <CPUTemp {cpuTemp} />
  </div>
  <div class="item-wrapper">
    <Storage {disk} />
  </div>
  <div class="item-wrapper">
    <SMARTCheck {smart} />
  </div>
</div>

<style>
  .sys-title {
    margin-bottom: 56px;
    display: flex;
    align-items: center;
  }
  .sys-title > span {
    flex: 1;
  }
  .title-actions {
    display: flex;
    gap: 16px;
    align-items: center;
  }
  .restart-groundseg {
    border-radius: 16px;
    background: var(--Gray-400, #5C7060);
    color: #FFF;

    text-align: center;
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    height: 64px;
    font-style: normal;
    font-weight: 300;
    line-height: 32px; /* 133.333% */
    letter-spacing: -1.44px;
    padding: 0 32px;
    min-width: 220px;
  }
  .restart-groundseg.secondary {
    background: var(--btn-secondary);
    min-width: 210px;
  }
  .restart-groundseg:disabled {
    opacity: 0.6;
    pointer-events: none;
  }
  @media (max-width: 900px) {
    .sys-title {
      align-items: flex-start;
      flex-direction: column;
      gap: 20px;
    }
    .title-actions {
      flex-wrap: wrap;
    }
  }
  .item-wrapper {
    margin-bottom: 32px;
  }
</style>
