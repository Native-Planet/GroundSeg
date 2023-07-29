<script>
  import { wide } from '$lib/stores/display'
  import './system.css'
  import SystemLogs from './SystemLogs.svelte'
  import Wireguard from './Wireguard.svelte'

  let tab = "system"
</script>

<div class="nav">
  <div class="tab" on:click={()=>tab="system"}>GroundSeg</div>
  <div class="tab" on:click={()=>tab="wireguard"}>Wireguard</div>
  <div class="tab" on:click={()=>tab="containers"}>Containers</div>
</div>
<div class="panel">
  {#if tab == "system"}
    <SystemLogs {wide} />
  {:else if tab == "wireguard"}
    <Wireguard {wide} />
  {/if}
</div>

<style>
  .nav {
    width: 1104px;
    display: flex;
    gap: 24px;
    justify-content: center;
    margin: 20px auto;
  }
  .tab {
    background: green;  
    font-family: var(--regular-font);
    font-size: 14px;
    line-height: 32px;
  }
  .panel {
    width: 1104px;
    max-width: 98vw;
    margin: auto;
    height: auto;
    border-radius: 16px;
  }
</style>
