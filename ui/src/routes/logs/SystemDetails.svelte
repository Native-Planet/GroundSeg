<script>
  import { structure } from '$lib/stores/websocket'
  $: usage = ($structure?.system?.usage) || {}
  $: ram = (usage?.ram) || [0,0]
  $: ramPercent = (ram[1]/ram[0] * 100).toFixed(2)
  $: cpu = (usage?.cpu) || 0
  $: cpuTemp = (usage?.cpu_temp) || 0
  $: disk = (usage?.disk) || [0,0,0]
  $: diskPercent = (disk[1]/disk[0] * 100).toFixed(2)
</script>


<div class="container">
  <!-- TITLE -->
  <div class="title">SYSTEM DETAILS</div>

  <!-- Contents -->
  <div class="wrapper">

    <!-- LEFT -->
    <div class="left">

      <!-- RAM -->
      <div class="row">
        <div class="label">RAM</div>
        <div class="ram-wrapper">
          <div class="ram-bg" style="width:{ramPercent}%"></div>
          <div class="ram-percent">{ramPercent}%</div>
        </div>
        <div class="ram-details">
          {(ram[1] / (1024 * 1024 * 1024)).toFixed(1)} GB / {(ram[0] / (1024 * 1024 * 1024)).toFixed(1)} GB
        </div>
      </div>

      <!-- STORAGE -->
      <div class="row">
        <div class="label">Storage</div>
        <div class="ram-wrapper">
          <div class="ram-bg" style="width:{diskPercent}%"></div>
          <div class="ram-percent">{diskPercent}%</div>
        </div>
        <div class="ram-details">
          {(disk[1] / (1024 * 1024 * 1024)).toFixed(1)} GB / {(disk[0] / (1024 * 1024 * 1024)).toFixed(1)} GB
        </div>
      </div>

      <div class="buttons">
        <button class="btn">Restart GroundSeg</button>
        <button class="btn">Update Linux</button>
      </div>
    </div>

    <!-- RIGHT -->
    <div class="right">
      <div class="row">
        <div class="cpu-label">CPU</div>
        <div class="cpu-details">TEMPERATURE</div>
        <div class="ram-wrapper">
          <div class="ram-bg" style="width:{cpuTemp}%"></div>
          <div class="ram-percent">{cpuTemp}&deg;C</div>
        </div>
        <div class="cpu-details">LOAD</div>
        <div class="ram-wrapper">
          <div class="ram-bg" style="width:{cpu}%"></div>
          <div class="ram-percent">{cpu}%</div>
        </div>
      </div>
    </div>

  </div>
</div>

<style>
  .wrapper {
    display: flex;
    gap: 80px;
  }
  .left {
    flex: 1;
    display: flex;
    flex-direction: column;
  }
  .right {
    flex: 1;
    display: flex;
    flex-direction: column;
  }
  .container {
    margin: 0;
  }
  .row {
    display: flex;
    flex-direction: column;
    font-size: 14px;
    gap: 12px;
    padding-bottom: 24px;
  }
  .label {
    font-family: var(--regular-font);
    font-size: 12px;
    padding-left: 10px;
    flex: 1;
  }
  .cpu-label {
    font-family: var(--regular-font);
    font-size: 12px;
    padding-left: 10px;
    flex: 1;
    margin-bottom: 10px;
  }
  .ram-wrapper {
    position: relative;
    background-color: var(--bg-modal);
    border-radius: 12px;
    border: solid 4px var(--btn-secondary);
    height: 24px;
  }
  .ram-bg {
    position: absolute;
    left: 0;
    top: 0;
    background: var(--btn-secondary);
    height: 100%;
  }
  .ram-percent {
    position: absolute;
    left: 16px;
    top: 0;
    font-family: var(--title-font);
    font-size: 16px;
    text-align: center;
    color: var(--text-card-color);
  }
  .ram-details {
    margin-top: -10px;
    padding-left: 10px;
    font-size: 18px;
    font-family: var(--title-font);
    color: var(--text-color);
  }
  .cpu-details {
    margin-bottom: -8px;
    padding-left: 10px;
    font-size: 18px;
    font-family: var(--title-font);
    color: var(--text-color);
  }
  .buttons {
    margin-top: 40px;
    display: flex;
    gap: 40px;
  }
  .btn {
    border-radius: 16px;
    flex: 1;
    background-color: var(--btn-secondary);
    color: var(--text-card-color);
    padding: 12px;
    font-family: var(--regular-font);
    font-size: 12px;
  }
</style>
