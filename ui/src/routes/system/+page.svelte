<script>
  import { wide } from '$lib/stores/display'
  import './system.css'
  import { shutdownModal, restartModal } from './store'
  import Connection from './Connection.svelte'
  import SystemDetails from './SystemDetails.svelte'
  import Power from './Power.svelte'
  import PowerModal from './PowerModal.svelte'
  import Netdata from './Netdata.svelte'
  import Logs from './Logs.svelte'
  //$: state = ($structure?.system?.updates?.linux?.state) || "updated"
</script>

<div class="panel">
  <Connection />
  <SystemDetails />
  <div class="wrapper">
    <Power />
  </div>
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
  .wrapper {
    display: flex;
    gap: 40px;
    margin-bottom: 40px;
  }
</style>
