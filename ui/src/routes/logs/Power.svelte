<script>
  import { structure } from '$lib/stores/websocket'
</script>

<div class="container wrapper">
    <div class="title">Power</div>
    <button class="power-btn">Shut Down</button>
    <button class="power-btn">Restart</button>
</div>

<style>
  .title {
    flex: 1;
    margin: 0;
  }
  .wrapper {
    display: flex;
    gap: 20px;
    margin: 0;
  }
  .power-btn {
    font-family: var(--regular-font);
    background: var(--btn-secondary);
    color: var(--text-card-color);
    height: 42px;
    font-size: 12px;
    border-radius: 16px;
    cursor: pointer;
    width: 140px;
  }
  .power-btn:hover {
    background: var(--bg-card);
  }
</style>
