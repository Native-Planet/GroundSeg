<script>
  import { createEventDispatcher } from 'svelte'
  const dispatch = createEventDispatcher()
  export let on = false
  export let loading = false
</script>

<div
  class="wrapper"
  class:loading={loading}
  class:highlight={on}
  on:click={()=>dispatch("click")}>
  <div class="text-wrapper">
    <div class="text">On</div>
    <div class="text">Off</div>
  </div>
  <div class="button {on ? "on" : "off"}"></div>
</div>

<style>
  .wrapper {
    position: relative;
    width: 80px;
    height: 40px;
    float: right;
    border-radius: 8px;
    background-color: var(--btn-secondary);
    color: var(--text-card-color);
    cursor: pointer;
  }
  .hightlight {
    background-color: var(--btn-primary);
  }
  .loading {
    opacity: .6;
    pointer-events: none;
  }
  .text-wrapper {
    display: flex;
    justify-content: center;
    align-items: center;
    height: 40px;
  }
  .text {
    flex: 1;
    text-align: center;
    font-size: 12px;
  }
  .button {
    position: absolute;
    background-color: var(--btn-special);
    width: 36px;
    height: 32px;
    top: 4px;
    border-radius: 6px;
  }
  .off {
    left: 4px;
    background-color: var(--bg-card);
  }
  .on {
    right: 4px;
  }
</style>
