<script>
  import { wide } from '$lib/stores/display'
  import './system.css'
  import Logs from './Logs.svelte'
</script>

<div class="panel">
  <Logs {wide} />
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
