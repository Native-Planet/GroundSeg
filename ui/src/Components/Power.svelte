<script>
  import Fa from 'svelte-fa'
  import { faPowerOff, faRotateRight } from '@fortawesome/free-solid-svg-icons/index.es'
</script>

<div class="control">
  <div class="control-title">Power</div>
  <button class="shutdown">
    <Fa icon={faPowerOff} size="0.85x" />
    <span>Shutdown</span>
  </button>
  <button class="restart">
    <Fa icon={faRotateRight} size="0.85x" />
    <span>Restart</span>
  </button>
</div>

<style>
  .control {
    display: flex;
    background: #0000006d;
    width: 300px;
    max-width: calc(100% - 80px);
    padding: 10px 40px 10px 40px;
    border-radius: 15px;
    font-size: 18px;
    gap: 12px;
    align-items: center;
  }
  .control-title {
    font-size: 16px;
    flex: 1;
  }
  button {
    font-size: 14px;
    background: none;
    border: none;
    color: inherit;
    font-family: inherit;
  }
  span {
    padding-left: 4px;
  }
  .shutdown {
    color: red;
  }
  .restart {
    color: orange;
  }
</style>
