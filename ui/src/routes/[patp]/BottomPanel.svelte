<script>
  import { showDeleteModal } from './store'
</script>
<div class="bottom-panel">
  <div class="btn">Logs</div>
  <div class="spacer"></div>
  <div class="btn">Rebuild</div>
  <div class="btn">Export</div>
  <div class="btn" on:click={()=>showDeleteModal.set(true)}>Delete</div>
</div>
<style>
  .bottom-panel {
    display: flex;
    position: absolute;
    bottom: 0;
    left: 32px;
    display: flex;
    gap: 8px;
    width: 85%;
  }
  .btn {
    color: var(--text-card-color);
    font-size: 12px;
    background-color: var(--fg-card);
    border-radius: 8px 8px 0 0;
    padding: 12px 42px;
    cursor: pointer;
  }
  .spacer {
    flex: 1;
  }
</style>
