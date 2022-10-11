<script>
  import Fa from 'svelte-fa'
  import {power} from '$lib/api'
  import { faPowerOff, faRotateRight } from '@fortawesome/free-solid-svg-icons/index.es'

  const restart = () => power.set('restart')
  const shutdown = () => power.set('shutdown')

</script>
<div class="control">
  <div class="control-title">Power</div>
  <button on:click={shutdown} class="shutdown">
    <Fa icon={faPowerOff} size="0.85x" />
    <span>Shutdown</span>
  </button>
  <button class="restart" on:click={restart}>
    <Fa icon={faRotateRight} size="0.85x" />
    <span>Restart</span>
  </button>
</div>

<style>
  .control {
    display: flex;
    background: #0000006d;
    width: 300px;
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
    cursor: pointer;
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
