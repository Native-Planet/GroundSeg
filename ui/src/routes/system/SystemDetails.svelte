<script>
  import { updateLinux, restartGroundSeg, setSwap, structure } from '$lib/stores/websocket'

  import Swap from './SysDetails/Swap.svelte'
  import Ram from './SysDetails/Ram.svelte'
  import Storage from './SysDetails/Storage.svelte'
  import CPULoad from './SysDetails/CPULoad.svelte'
  import CPUTemp from './SysDetails/CPUTemp.svelte'

  $: usage = ($structure?.system?.usage) || {}
  $: swap = (usage?.swap) || 0
  $: ram = (usage?.ram) || [0,0]
  $: disk = (usage?.disk) || [0,0,0]
  $: cpu = (usage?.cpu) || 0
  $: cpuTemp = (usage?.cpu_temp) || 0
</script>


<div class="container">
  <div class="title">SYSTEM DETAILS</div>
  <Swap {swap}/>
  <Ram {ram} />
  <Storage {disk} />
  <CPULoad {cpu} />
  <CPUTemp {cpuTemp} />
</div>

<style>
  .title {
    margin-bottom: 24px;
  }
  .container {
    margin: 0;
  }
</style>
