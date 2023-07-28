<script>
  import { wide } from '$lib/stores/display'
  import './system.css'
  import SystemLogs from './SystemLogs.svelte'
  import Wireguard from './SystemLogs.svelte'
</script>

<div class="panel">
  <!--SystemLogs {wide} /-->
  <Wireguard {wide} />
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
</style>
