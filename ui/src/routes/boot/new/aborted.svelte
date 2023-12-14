<script>
  import { createEventDispatcher, onMount } from 'svelte'
  import { goto } from '$app/navigation';
  import { resetNewShip } from '$lib/stores/websocket'
  export let name = ""
  export let error = ""
  const dispatch = createEventDispatcher()
  onMount(()=>dispatch("emit"))
</script>

<div class="error">Error: {error}</div>
<div class="reset" on:click={resetNewShip}>Try Again</div>

<style>
  .error {
    padding: 24px;
    font-size: 20px;
    font-weight: 400;
    color: red;
  }
  .reset {
    margin-top: 20px;
    font-size: 14px;
    font-weight: 300;
    text-decoration: underline;
    cursor: pointer;
  }
</style>
