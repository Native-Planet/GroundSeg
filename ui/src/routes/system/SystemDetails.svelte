<script>
  import { structure } from '$lib/stores/websocket'
  export let wide
  $: usage = ($structure?.system?.usage) || {}
  $: ram = (usage?.ram) || 0
  $: cpu = (usage?.cpu) || 0
  $: cpu_temp = (usage?.cpu_temp) || 0
  $: disk = (usage?.disk) || 0
</script>

<div class="container {wide ? "wide" : "slim"}">
  <div class="title">System Details</div>
  <div class="wrapper">
    <div class="row">
      <div class="label">RAM</div>
      <div class="value">{disk}</div>
    </div>
    <div class="row">
      <div class="label">CPU Temperature</div>
      <div class="value">{cpu_temp}</div>
    </div>
    <div class="row">
      <div class="label">CPU Load</div>
      <div class="value">{cpu}</div>
    </div>
    <div class="row">
      <div class="label">Storage</div>
      <div class="value">{disk}</div>
    </div>
    <div class="storage-wrapper">
      <div class="used"></div>
      <div class="remainder"></div>
    </div>
  </div>
</div>

<style>
  .wide {
    width: 992px;
    max-width: 98vw;
    padding: 56px;
  }
  .slim {
    width: 100vw;
  }
  .wrapper {
    display: flex;
    flex-direction: column;
    gap: 20px;
  }
  .row {
    display: flex;
    font-size: 14px;
  }
  .label {
    flex: 1;
  }
  .storage-wrapper {
    display: flex;
    background-color: var(--btn-secondary);
    border-radius: 16px;
    height: 42px;
    border: solid 8px var(--btn-secondary);
    gap: 8px;
  }
  .used {
    width: 34%;
    height: 100%;
    background-color: var(--bg-warning);
    border-radius: 16px;
  }
  .remainder {
    flex: 1;
    height: 100%;
    background-color: var(--bg-modal);
    border-radius: 16px;
  }
</style>
