<script>
  import { structure } from '$lib/stores/websocket'
  export let wide
</script>

<div class="container {wide ? "wide" : "slim"}">
    <div class="title">Power</div>
    <button class="power-btn">Shut Down</button>
    <button class="power-btn">Restart</button>
</div>

<style>
  .wide {
    width: 992px;
    max-width: 98vw;
    padding: 56px;
  }
  .slim {
    width: 100vw;
  }
  .title {
    flex: 1;
    margin: 0;
  }
  .container {
    display: flex;
    gap: 20px;
  }
  .power-btn {
    background: var(--btn-secondary);
    color: var(--text-card-color);
    height: 42px;
    font-size: 14px;
    width: 200px;
    border-radius: 16px;
    cursor: pointer;
  }
</style>
