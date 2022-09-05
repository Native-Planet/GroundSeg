<script>
  import { onMount, onDestroy } from 'svelte'
  import { settings } from '$lib/components'
  import { api } from '$lib/api'
  import { page } from '$app/stores';

  let info, opened

  const update = () => {
    if (opened) {
      fetch(api + "/settings").then(r => r.json()).then(d => info = d)
      setTimeout(update, 1000)
  }}

  onMount( ()=> {opened = true; update()})
  onDestroy(() => opened = false)

</script>

<svelte:component this={settings.logo} t="System settings" />

<div class="container">

  <div class="panel">
    <svelte:component this={settings.sysInfo} {info} />
    <svelte:component this={settings.network} />
  </div>

  <div class="panel">
    <svelte:component this={settings.anchor} {info} />
    <svelte:component this={settings.minIO} {info} />
    <svelte:component this={settings.logs} />
    <svelte:component this={settings.power} />
  </div>

</div>

<style>
  .container {
    padding: 20px;
    width: 772px;
    max-width: calc(100vw - 40px);
    overflow: auto;
    display: flex;
    flex-wrap: wrap;
    gap: 12px;
    -ms-overflow-style: none;
    scrollbar-width: none;
  }
  .container::-webkit-scrollbar {
    display: none;
  }
  .panel {
    display: flex;
    flex-direction: column;
    gap: 12px;
    width: 380px;
  }
</style>








