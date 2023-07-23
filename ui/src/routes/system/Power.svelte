<script>
  import { structure } from '$lib/stores/websocket'
</script>

<div class="container">
    <div class="wrapper">
      <div class="title">POWER</div>
      <button class="power btn">Shut Down</button>
      <button class="restart btn">Restart</button>
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
