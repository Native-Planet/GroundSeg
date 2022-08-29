<script>
  import Logo from '/src/Components/Buttons/Logo.svelte'
</script>

<div class="head">
  <Logo />
  <div class="title">Settings</div>
</div>
<div class="container">

  <div class="sys">
    <div class="sys-title">System Information</div>
    <div class="ram">RAM 12.5%</div>
    <div class="storage">Storage 123 GB / 133 GB</div>
    <div class="temp">CPU Temp 36 &deg C</div>
  </div>

  <div class="setting">
    <div class="setting-title">WiFi/Connection Settings</div>
  </div>

  <div class="setting">
    <div class="setting-title">Anchor Settings</div>
  </div>

  <div class="setting">
    <div class="setting-title">Bitcoin Location</div>
  </div>

  <div class="control">
    <div class="control-title">Power</div>
    <div class="power-buttons">
      <button class="shutdown">Shutdown</button>
      <button class="restart">Restart</button>
    </div>
  </div>

  <div class="logs">
    <div class="logs-title">Export Logs</div>
    <a class="view" href="#">View</a>
    <button class="export">Export</button>
  </div>

</div>

<style>
  .head {
    display: flex;
    align-items: end;
  }
  .container {
    padding: 20px;
    width: 480px;
    max-width: calc(100vw - 40px);
  }
  .title {
    font-size: 24px;
    padding: 0 0 4px 16px;
  }
  .setting {
    padding: 8px 0 8px 0;
  }
  .setting-title {
    font-size: 18px;
  }
  .sys {
  padding-bottom: 12px;
  }
  .sys-title {
    font-size: 18px;
  }

</style>








