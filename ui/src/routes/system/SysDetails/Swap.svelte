<script>
  import { structure, setSwap } from '$lib/stores/websocket'

  import Fa from 'svelte-fa'
  import { faMinus, faPlus } from '@fortawesome/free-solid-svg-icons';

  export let swap
</script>

<div class="panel">
  <div class="left">
    <div class="swap-title">Use Storage for RAM</div>
    <div class="swap-subtitle">Increase your RAM by borrowing from your Storage Drive</div>
  </div>
  <div class="right">
    <button class="btn" disabled={swap==0} on:click={()=>setSwap(swap-1)}>
      <Fa icon={faMinus} size="1x" />
    </button>
    <div class="val">{swap} GB</div>
    <button class="btn" on:click={()=>setSwap(swap+1)}>
      <Fa icon={faPlus} size="1x" />
    </button>
  </div>
</div>

<style>
  .panel {
    display: flex;
  }
  .left {
    flex: 1;
  }
  .swap-title {
    font-size: 16px;
    font-family: var(--regular-font);
    margin-bottom: 4px;
  }
  .swap-subtitle {
    font-size: 12px;
    font-family: var(--regular-font);
  }
  .right {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 24px;
  }
  .btn {
    background: var(--btn-secondary);
    width: 64px;
    color: var(--text-card-color);
    text-align: center;
    height: 42px;
    line-height: 38px;
    border-radius: 8px;
    font-size: 20px;
    cursor: pointer;
    padding-bottom: 4px;
  }
  .btn:disabled {
    opacity: 0.6;
    pointer-events: none;
  }
  .val {
    font-size: 24px;
  }
</style>
