<script>
  import { setSwap } from '$lib/stores/websocket'
  import { structure } from '$lib/stores/data'

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
    color: var(--NP_Black, #313933);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    line-height: normal;
    letter-spacing: -1.44px;
  }
  .swap-subtitle {
    color: var(--NP_Black, #313933);
    font-family: Inter;
    font-size: 16px;
    font-style: normal;
    font-weight: 300;
    line-height: 24px; /* 150% */
    letter-spacing: -0.96px;
  }
  .right {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 24px;
  }
  .btn {
    background: var(--btn-secondary);
    width: 72px;
    color: var(--text-card-color);
    text-align: center;
    height: 56px;
    line-height: 38px;
    border-radius: 16px;
    font-size: 20px;
    cursor: pointer;
    padding-bottom: 4px;
  }
  .btn:disabled {
    opacity: 0.6;
    pointer-events: none;
  }
  .val {
    color: #000;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    line-height: 24px; /* 100% */
    letter-spacing: -1.44px;
  }
</style>
