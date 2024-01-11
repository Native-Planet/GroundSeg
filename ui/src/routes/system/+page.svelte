<script>
  import './system.css'
  import { showLogs } from './store'

  import { structure, URBIT_MODE } from '$lib/stores/data'
  import { wide } from '$lib/stores/display'

  import LinuxUpdate from './LinuxUpdate.svelte'
  import Connection from './Connection.svelte'
  import SystemDetails from './SystemDetails.svelte'
  import Power from './Power.svelte'
  import Logs from './Logs.svelte'
  import Penpai from './Penpai.svelte'
  import Support from './Support.svelte'

  import { printMounts } from '$lib/stores/websocket'

  $: state = ($structure?.system?.updates?.linux?.state) || "updated"
</script>

<div class="panel">
  <!--
  <button on:click={printMounts}>Print Mounts</button>
  -->
  {#if state != "updated"}
    <LinuxUpdate />
  {/if}
  <Connection />
  <SystemDetails />
  <Power />
  {#if !$URBIT_MODE}
    <Logs />
  {/if}
  <Penpai />
  <Support />
</div>

<style>
  .panel {
    width: 1104px;
    max-width: 98vw;
    margin: auto;
    height: auto;
    border-radius: 16px;
    display :flex;
    flex-direction: column;
    gap: 20px;
  }
</style>
