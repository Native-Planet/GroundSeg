<script>
  import { updateLinux, restartGroundSeg, setSwap } from '$lib/stores/websocket'
  import { connected, structure } from '$lib/stores/data'

  import { afterUpdate } from 'svelte'

  import Swap from './SysDetails/Swap.svelte'
  import Ram from './SysDetails/Ram.svelte'
  import Storage from './SysDetails/Storage.svelte'
  import CPULoad from './SysDetails/CPULoad.svelte'
  import CPUTemp from './SysDetails/CPUTemp.svelte'

  $: usage = ($structure?.system?.info?.usage) || {}
  $: swap = (usage?.swap) || 0
  $: ram = (usage?.ram) || [0,0]
  $: disk = (usage?.disk) || {};
  $: cpu = (usage?.cpu) || 0
  $: cpuTemp = (usage?.cpu_temp) || 0

  let restarting = false
  let dead = false
  let success = false

  const handleGroundSegRestart = () => {
    restarting = true
    restartGroundSeg()
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
    <button on:click={handleGroundSegRestart} class="restart-groundseg" disabled={restarting}>
      {#if success}
        GroundSeg Restarted!
      {:else}
        {restarting ? "GroundSeg Restarting..." : "Restart GroundSeg"}
      {/if}
    </button>
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
  .sys-title > button {
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
    padding: 0 48px;
  }
  .sys-title > button:disabled {
    opacity: 0.6;
    pointer-events: none;
  }
  .item-wrapper {
    margin-bottom: 32px;
  }
</style>
