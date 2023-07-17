<script>
  import { createEventDispatcher } from 'svelte'
  export let view
  export let advanced = false
  const dispatch = createEventDispatcher()
</script>

<button on:click={dispatch('click', !view)}>
  <img style="{advanced ? "margin-top: 6px" : "margin-left:12px"};" class="eye" src={view ? "/eye-closed.svg" : "/eye-open.svg"} alt="eye" />
</button>

<style>
button {
  appearance: none;
  background: none;
  border: none;
  padding: 0;
  margin: 0;
  height: 32px;
}
.eye {
  opacity: .8;
  cursor: pointer;
}
</style>
