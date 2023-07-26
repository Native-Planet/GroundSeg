<script>
  import { structure, registerServiceAgain } from '$lib/stores/websocket';
  export let patp

  $: ship = ($structure?.urbits?.[patp]) || {}
  $: memUsage = (ship?.memUsage) || 0
  $: diskUsage = (ship?.diskUsage) || 0
  $: loom = (ship?.loomSize) || 0
  $: svcRegStatus = "ok" //(ship?.serviceRegistrationStatus) || "ok"

  $: chars = (patp.replace(/-/g,"").length) || 0
  $: shipClass = (chars == 3 ? "GALAXY" : chars == 6 ? "STAR" : chars == 12 ? "PLANET" : chars > 12 ? "MOON" : "UNKNOWN") || "ERROR"


</script>

<div class="header">
  <div class="patp-wrapper">
    <div class="ship-class">{shipClass}</div>
    <div class="patp">{patp.toUpperCase()}</div>
    <div class="quick-panel">
      <button
        disabled={svcRegStatus != "ok"}
        on:click={()=>registerServiceAgain(patp)}
        class="btn svc-register">{svcRegStatus == "creating" ? "Registering" : "Reregister Services"}
      </button>
      <button class="btn rebuild-container">Rebuild Container</button>
    </div>
  </div>
  <div class="settings-wrapper">
    <div class="settings">
      <div class="settings-text">RAM</div>
      <div class="settings-val">{parseInt(memUsage/(1024*1024))} MB / {loom} MB</div>
    </div>
    <div class="settings">
      <div class="settings-text">DISK</div>
      <div class="settings-val">{(diskUsage/(1024*1024)).toFixed(2)} MB</div>
    </div>
  </div>
</div>

<style>
  .header {
    background-color: var(--bg-card);
    color: var(--text-card-color);
    position: absolute;
    height: 150px;
    width: calc(1173px - 150px);
    max-width: calc(100vw - 150px);
    left: 150px;
    border-radius: 16px 16px 0 0;
    position: relative;
  }
  .patp-wrapper {
    background: var(--fg-card);
    position: absolute;
    top: 16px;
    left: 16px;
    width: 567px;
    height: 134px;
    border-radius: 16px;
  }
  .ship-class {
    font-family: var(--title-font);
    margin: 20px 0 0 20px;
  }
  .patp {
    font-family: var(--title-font);
    margin-left: 20px;
    font-size: 32px;
  }
  .settings-wrapper {
    background: var(--fg-card);
    font-family: var(--title-font);
    position: absolute;
    display: flex;
    flex-direction: column;
    gap: 8px;
    width: 320px;
    padding: 24px 32px;
    right: 0;
    border-radius: 0px 16px;
  }
  .settings {
    display: flex;
    font-size: 24px;
  }
  .settings-text {
    flex: 1;
  }
  .quick-panel {
    display: flex;
    margin: 12px 20px;
    gap: 20px;
  }
  .btn {
    width: 30%;
    font-family: var(--regular-font);
    font-size: 12px;
    line-height: 32px;
    background-color: var(--btn-secondary);
    color: var(--text-card-color);
    border-radius: 8px;
    cursor: pointer;
  }
  .btn:hover {
    background: var(--bg-card);
  }
  .btn:disabled {
    pointer-events: none;
    opacity: .6;
  }
</style>
