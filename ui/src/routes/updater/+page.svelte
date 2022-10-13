<script>
  import { onMount, onDestroy } from 'svelte'
  import { api } from '$lib/api'

  let heartBeat = '', 
    opened =  true

  const checkHeartBeat = () => {
    if (opened) {
    let u = $api + "/updater"
    fetch(u).then(r => r.json()).then(d => heartBeat = d)
    setTimeout(checkHeartBeat, 1000)
    }}

  onMount(() => checkHeartBeat())
  onDestroy(() => opened = false)

  
</script>

<div class="container">

  {#if heartBeat == 'updating'}
    <div class="title updating">Groundseg is updating...</div>
    <div class="warning">Do not turn off your Native Planet device</div>
  {/if}

  {#if heartBeat == 'live'}
    <div class="title">Your Groundseg has been updated!</div>
    <button on:click={()=>window.location.href = "/"}>Let's go</button>
  {/if}

</div>

<style>
  .container {
    width: 500px;
    max-width: calc(100vw - 120px);
    height: 120px;
    padding: 60px;
    text-align: center;
  }
  .title {
    font-size: 24px;
    line-height: 60px;
  }
  .updating {
    animation: breathe 10s infinite;
  }
  .warning {
    font-size: 16px;
    font-style: italic;
    line-height: 60px;
  }

  button {
    appearance: none;
    background: var(--action-color);
    color: inherit;
    padding: 8px;
    border-radius: 8px;
    width: 120px;
    border: none;
    cursor: pointer;
    font-family: inherit;
    margin-top: 24px;
  }

  @keyframes breathe {
    0% {opacity: 1}
    50% {opacity: .4}
    100% {opacity: 1}
  }

</style>
