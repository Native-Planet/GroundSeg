<script>
	import { createEventDispatcher } from 'svelte'
	export let view
	const dispatch = createEventDispatcher()
</script>
<button on:click={dispatch('click', !view)}>
	<img class="eye" src={view ? "/eye-closed.svg" : "/eye-open.svg"} alt="eye" />
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
  height: 32px;
  opacity: .8;
  margin-left: 12px;
  cursor: pointer;
}
</style>
