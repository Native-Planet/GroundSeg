
<div class="body">
  <div class="bottom-panel">
    <div class="btn">Logs</div>
    <div class="btn">Shutdown</div>
    <div class="btn">Advanced</div>
    <div class="btn">Export/Delete</div>
  </div>
</div>

<style>
  .body {
    background-color: var(--bg-card);
    position: absolute;
    bottom: 0;
    height: calc(793px - 150px);
    width: 100%;
    max-width: 100vw;
    border-radius: 16px 0 120px 16px;
  }
  .bottom-panel {
    position: absolute;
    bottom: 0;
    left: 32px;
    display: flex;
    gap: 8px;
  }
  .btn {
    color: var(--text-card-color);
    font-size: 16px;
    background-color: var(--fg-card);
    border-radius: 16px 16px 0 0;
    padding: 16px 32px;
  }
</style>
