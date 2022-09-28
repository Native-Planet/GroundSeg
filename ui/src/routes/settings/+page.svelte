<script>
  import { onMount, onDestroy } from 'svelte'
  import { settings } from '$lib/components'
  import { power, api } from '$lib/api'
  import { page } from '$app/stores'

  let info, opened

  const update = () => {
    if (opened) {
      fetch(api + "/settings").then(r => r.json()).then(d => info = d)
      setTimeout(update, 1000)
  }}

  onMount( ()=> {opened = true; update()})
  onDestroy(() => opened = false)

  power.set(null)

</script>

<svelte:component this={settings.logo} t="System settings" />

<div class="wrapper">
<div class="content">

  <div class="panel">
    <svelte:component this={settings.sysInfo} {info} />
    <svelte:component this={settings.minIO} {info} />
    <svelte:component this={settings.network} {info} />
    <svelte:component this={settings.power} />
  </div>

  <div class="panel">
    <svelte:component this={settings.anchor} {info} />
    <svelte:component this={settings.exportLogs} />
    <svelte:component this={settings.contact} />
  </div>

</div>
</div>
<style>
  .content {
    padding: 20px;
    width: 772px;
    overflow: auto;
    max-width: calc(100vw - 40px);
    max-height: 70vh;
    display: flex;
    flex-wrap: wrap;
    gap: 12px;
    -ms-overflow-style: none;
    scrollbar-width: none;
  }
  .content::-webkit-scrollbar {
    display: none;
  }
  .panel {
    display: flex;
    flex-direction: column;
    gap: 12px;
    width: 380px;
  }
</style>








