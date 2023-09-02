<script>
  import { createEventDispatcher, onMount } from 'svelte'
  import { goto } from '$app/navigation';
  import { resetNewShip } from '$lib/stores/websocket'

  export let name = ""

  const dispatch = createEventDispatcher()
  onMount(()=>dispatch("emit"))

  // go to pier settings
  const handleClick = () => {
    handleReset()
    goto("/"+name)
  }
  // reset transitions
  const handleReset = () => {
    resetNewShip()
  }
</script>

<div class="text">Boot Complete</div>
<div class="buttons">
  <button class="btn" on:click={handleClick}>Settings</button>
  <a class="btn" href="#" on:click={handleReset} target="_blank">Visit URL</a>
</div>

<style>
  .text {
    padding: 24px;
    font-size: 20px;
    font-weight: 400;
    color: var(--text-color);
  }
  .buttons {
    display: flex;
    width: 280px;
    gap: 20px;
    text-align: center;
    align-items: center;
  }
  button {
    background: var(--btn-secondary);
  }
  a {
    background: var(--btn-primary);
  }
  .btn {
    font-size: 12px;
    height: 42px;
    line-height: 42px;
    color: var(--text-card-color);
    text-decoration: none;
    flex: 1;
    border-radius: 12px;
    cursor: pointer;
  }
</style>
