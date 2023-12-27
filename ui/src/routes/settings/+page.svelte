<script>
  import './system.css'
  import { showLogs } from './store'

  import { structure } from '$lib/stores/websocket'
  import { wide } from '$lib/stores/display'

  import LinuxUpdate from './LinuxUpdate.svelte'
  import Connection from './Connection.svelte'
  import SystemDetails from './SystemDetails.svelte'
  import Power from './Power.svelte'
  import Logs from './Logs.svelte'
  import Penpai from './Penpai.svelte'
  import Support from './Support.svelte'

  $: state = ($structure?.system?.updates?.linux?.state) || "updated"
</script>

<div class="panel">
  {#if state != "updated"}
    <LinuxUpdate />
  {/if}
  <Connection />
  <SystemDetails />
  <Power />
  <Logs />
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
