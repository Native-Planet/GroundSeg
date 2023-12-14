<script>
  import './system.css'
  import { showLogs } from './store'

  import { structure } from '$lib/stores/data'
  import { wide } from '$lib/stores/display'

  import LinuxUpdate from './linuxupdate.svelte'
  import Connection from './connection.svelte'
  import SystemDetails from './systemdetails.svelte'
  import Power from './power.svelte'
  import Logs from './logs.svelte'
  import Penpai from './penpai.svelte'
  import Support from './support.svelte'

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
