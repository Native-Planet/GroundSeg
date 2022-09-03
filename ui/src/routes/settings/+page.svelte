<script>
  import { onMount, onDestroy } from 'svelte'
  import { url } from '/src/Scripts/server'
  import { page } from '$app/stores';
  import Logo from '/src/Components/Buttons/Logo.svelte'
  import SysInfo from '/src/Components/SysInfo.svelte'
  import Power from '/src/Components/Power.svelte'
  import Network from '/src/Components/Network.svelte'
  import MinIO from '/src/Components/MinIO.svelte'
  import Anchor from '/src/Components/Anchor.svelte'
  import Logs from '/src/Components/Logs.svelte'

  let info, opened

  const update = () => {
    if (opened) {
      fetch(url + "/settings").then(r => r.json()).then(d => info = d)
      setTimeout(update, 1000)
  }}

  onMount( ()=> {opened = true; update()})
  onDestroy(() => opened = false)

</script>

<Logo t="System Settings" />
<div class="container">
  <div class="panel">
    <SysInfo {info} />
    <Network />
  </div>
  <div class="panel">
    <Anchor />
    <MinIO />
    {#if info}
      <Logs />
    {/if}
    <Power />
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








