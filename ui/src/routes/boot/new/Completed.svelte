<script>
  import { createEventDispatcher, onMount } from 'svelte'
  import { goto } from '$app/navigation';
  import { structure, resetNewShip } from '$lib/stores/websocket'

  export let name = ""

  $: url = ($structure?.urbits?.[name]?.info?.url) || "#"

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
  <a class="btn" href={url} on:click={handleReset} target="_blank">Visit URL</a>
</div>
<div class="reset" on:click={handleReset}>Boot Another Ship</div>

<style>
  .text {
    padding: 24px;
    font-size: 20px;
    font-weight: 400;
    color: var(--text-color);
  }
  .buttons {
    display: flex;
    gap: 16px;
    text-align: center;
    align-items: center;
    height: 65px;
    justify-content: center;
  }
  button {
    background: var(--btn-secondary);
  }
  a {
    background: var(--btn-primary);
  }
  .btn {
    color: #FFF;
    text-align: center;
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    line-height: 32px; /* 133.333% */
    letter-spacing: -1.44px;
    padding: 16px 48px 17px 48px;
    border-radius: 16px;
  }
  .reset {
    margin-top: 40px;
    font-size: 14px;
    font-weight: 300;
    text-decoration: underline;
    cursor: pointer;
  }
</style>
