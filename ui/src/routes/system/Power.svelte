<script>
  import { structure } from '$lib/stores/websocket'
  import { shutdownModal, restartModal } from './store'
</script>

<div class="container">
    <div class="wrapper">
      <div class="title">POWER</div>
      <button class="power btn" on:click={()=>shutdownModal.set(true)}>Shut Down</button>
      <button class="restart btn" on:click={()=>restartModal.set(true)}>Restart</button>
    </div>
</div>

<style>
  .title {
    margin: 0;
  }
  .wrapper {
    display: flex;
    gap: 20px;
    margin: 0;
  }
  .btn {
    flex: 1;
    font-family: var(--regular-font);
    color: var(--text-card-color);
    height: 36px;
    font-size: 12px;
    border-radius: 16px;
    cursor: pointer;
  }
  .btn:hover {
    background: var(--bg-card);
  }
  .restart {
    background: var(--btn-secondary);
  }
  .power {
    background: var(--btn-secondary);
  }
</style>
