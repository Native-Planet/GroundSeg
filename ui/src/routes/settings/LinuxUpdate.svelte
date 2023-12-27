<script>
  import { updateLinux } from '$lib/stores/websocket'
</script>

<div class="container">
  <div class="title">UPDATE</div>
  <div class="subtitle">There is a new version of Linux. Please update your system to continue.</div>
  <button class="btn" on:click={updateLinux}>Update</button>
</div>

<style>
  .container {
    margin: 0;
    gap: 24px;
    align-items: center;
    background-color: var(--bg-warning);
    display: flex;
    flex-direction: column;
    gap: 8px;
    align-items: start;
    padding: 20px 40px;
  }
  .title {
    margin: 0;
  }
  .subtitle {
    font-size: 14px;
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
    background: #000;
  }
</style>
