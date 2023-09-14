<script>
  import { showDeleteModal } from './store'
  import LogsDrawer from './LogsDrawer.svelte'
  import { openModal } from 'svelte-modals'
  export let patp
</script>
<div class="bottom-panel">
  <button 
    class="btn" 
    on:click={()=>openModal(LogsDrawer,{"patp":patp})}>
    Logs
  </button>
  <div class="spacer"></div>
  <div class="btn rebuild">Rebuild</div>
  <div class="btn">Export</div>
  <div class="btn" on:click={()=>showDeleteModal.set(true)}>Delete</div>
</div>
<style>
  .bottom-panel {
    display: flex;
    bottom: 0;
    gap: 12px;
    width: 85%;
  }
  .btn {
    color: var(--NP_White, #F8F8F6);
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    line-height: 24px; /* 100% */
    letter-spacing: -1.44px;

    color: var(--text-card-color);
    background-color: var(--text-color);
    cursor: pointer;
    border-radius: 16px 16px 0px 0px;

    display: inline-flex;
    padding: 16px 32px;
    justify-content: center;
    align-items: center;
    gap: 8px;
  }
  .rebuild {
    background-color: var(--fg-card);
  }
  .spacer {
    flex: 1;
  }
</style>
