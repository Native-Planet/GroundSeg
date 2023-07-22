<script>
  export let patp
</script>

<div class="header">
  <div class="patp-wrapper">
    <div class="ship-class">Planet</div>
    <div class="patp">{patp.toUpperCase()}</div>
  </div>
  <div class="settings-wrapper">
    <div class="settings">
      <div class="settings-text">RAM</div>
      <div class="settings-val">1012 / 2048 MB</div>
    </div>
    <div class="settings">
      <div class="settings-text">DISK</div>
      <div class="settings-val">3.2 GB</div>
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
</style>
