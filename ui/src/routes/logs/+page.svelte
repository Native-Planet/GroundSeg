<script>
  import { wide } from '$lib/stores/display'
  import './system.css'
  import Connection from './Connection.svelte'
  import SystemDetails from './SystemDetails.svelte'
  import Power from './Power.svelte'
  import Netdata from './Netdata.svelte'
  import Logs from './Logs.svelte'
</script>

<div class="panel">
  <SystemDetails />
  <Connection />
  <div class="wrapper">
    <Power />
    <Netdata />
  </div>
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
    gap: 40px;
  }
  .wrapper {
    display: flex;
    gap: 40px;
    margin-bottom: 40px;
  }
  .left {
    display: flex;
    flex: 1;
    flex-direction: column;
    gap: 40px;
  }
  .right {
    display: flex;
    flex: 1;
    flex-direction: column;
    gap: 40px;
  }
</style>
