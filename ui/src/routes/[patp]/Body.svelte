<script>
  import { structure, toggleDevMode } from '$lib/stores/websocket'
  export let patp

  $: ship = ($structure?.urbits?.[patp]?.info)
  $: devMode = (ship?.devMode) || false

</script>
<div class="body">

  <!-- Dev Mode -->
  <div class="section">
    <div class="section-left">
      <div class="title">Developer Mode</div>
      <div class="description">This enables your ship to be able to debug from another computer</div>
    </div>
    <div class="section-right">
      <div on:click={()=>toggleDevMode(patp)}>{devMode ? "on" : "off"}</div>
    </div>
  </div>

  <!-- Urbit Ship Status -->
  <div class="section">
    <div class="section-left">
      <div class="title">Remember Urbit Boot Status</div>
      <div class="description">This enables your ship to autoboot after a device restart</div>
    </div>
    <div class="section-right">
      checkbox
    </div>
  </div>

  <!-- Loom -->
  <div class="section">
    <div class="section-left">
      <div class="title">Loom Size</div>
      <div class="description">Loom description</div>
    </div>
    <div class="section-right">
      slider
    </div>
  </div>

  <!-- Pack & Meld -->
  <div class="section">
    <div class="section-left">
      <div class="title">Pack & Meld</div>
      <div class="description">This function will refragement your ship's memory capacity, optimizing its performance. We recommend scheduling these once a week</div>
    </div>
    <div class="section-right">
      Start Button, calendar
    </div>
  </div>

  <!-- Startram Connectivity -->
  <div class="section">
    <div class="section-left">
      <div class="title">StarTram</div>
      <div class="description">Allow toggling StarTram</div>
    </div>
    <div class="section-right">
      on/off
    </div>
  </div>

  <div>url</div>
  <div>MinIO</div>
  <div>vere version</div>
  <div>minio custom domain</div>
  <div>urbit custom domain</div>

  <div class="bottom-panel">
    <div class="btn">Logs</div>
    <div class="btn">Shutdown</div>
    <div class="btn">Export/Delete</div>
  </div>
</div>

<style>
  .body {
    background-color: var(--bg-card);
    position: absolute;
    bottom: 0;
    height: calc(743px - 150px);
    width: calc(100% - 40px);
    padding: 0 20px;
    max-width: 100vw;
    border-radius: 16px 0 120px 16px;
    color: var(--text-card-color);
    display: flex;
    flex-direction: column;
    gap: 20px;
  }
  .bottom-panel {
    position: absolute;
    bottom: 0;
    left: 32px;
    display: flex;
    gap: 8px;
  }
  .section {
    display: flex;
  }
  .section-left {
    flex: 1;
  }
  .section-right {
    flex: 1;
    text-align: right;
  }
  .btn {
    color: var(--text-card-color);
    font-size: 16px;
    background-color: var(--fg-card);
    border-radius: 16px 16px 0 0;
    padding: 16px 32px;
  }
  .title {
    font-size: 16px;
  }
  .description {
    font-size: 12px;
  }
</style>
