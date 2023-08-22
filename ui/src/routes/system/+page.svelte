<script>
  import './system.css'

  import { shutdownModal, restartModal } from './store'
  import { structure } from '$lib/stores/websocket'
  import { wide } from '$lib/stores/display'

  import LinuxUpdate from './LinuxUpdate.svelte'
  import Connection from './Connection.svelte'
  import SystemDetails from './SystemDetails.svelte'
  import Power from './Power.svelte'
  import PowerModal from './PowerModal.svelte'
  import Logs from './Logs.svelte'
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
  <Support />
</div>

{#if $shutdownModal}
  <PowerModal info="shutdown" />
{:else if $restartModal}
  <PowerModal info="restart" />
{/if}

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
