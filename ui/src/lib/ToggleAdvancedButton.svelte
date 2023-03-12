<script>
  import { createEventDispatcher } from 'svelte'
  export let advanced, style = "float:right;"
  const dispatch = createEventDispatcher()
</script>
<button class="advanced" on:click={()=>dispatch('click')} style={style}>
  {advanced ? "Back" : "Advanced Options"}
</button>

<style>
  .advanced {
    font-family: inherit;
    color: inherit;
    border: solid 1px #ffffff4d;
    border-radius: 8px;
    margin-top: 12px;
    padding: 6px;
    width: 160px;
    font-size: 11px;
  }
  .advanced:hover {
    background: #008CFB;
    border-color: #008CFB;
    cursor: pointer;
  }
</style>
