<script>
  import { showLogs } from './store'
  import LogsDrawer from './LogsDrawer.svelte'
  import { openModal } from 'svelte-modals'
</script>

<div class="container">
  <div class="title">SYSTEM LOGS</div>
  <div class="spacer"></div>
  <button class="btn" on:click={()=>openModal(LogsDrawer,{"title":"System Logs"})}>View</button>
</div>

<style>
  .container {
    margin: 0;
    display: flex;
    gap: 24px;
    align-items: center;
  }
  .title {
    margin: 0;
  }
  .spacer {
    flex: 1;
  }
  .btn {
    font-family: var(--regular-font);
    color: var(--text-card-color);
    height: 42px;
    font-size: 12px;
    font-weight: 600;
    border-radius: 12px;
    cursor: pointer;
    padding: 0 36px;
    background: var(--btn-secondary);
  }
</style>
