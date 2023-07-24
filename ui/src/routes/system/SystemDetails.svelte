<script>
  import { updateLinux, restartGroundSeg, setSwap, structure } from '$lib/stores/websocket'
  $: usage = ($structure?.system?.usage) || {}
  $: ram = (usage?.ram) || [0,0]
  $: ramPercent = (ram[1]/ram[0] * 100).toFixed(2)
  $: cpu = (usage?.cpu) || 0
  $: cpuTemp = (usage?.cpu_temp) || 0
  $: disk = (usage?.disk) || [0,0,0]
  $: diskPercent = (disk[1]/disk[0] * 100).toFixed(2)

  $: state = ($structure?.system?.updates?.linux?.state) || "updated"

  $: swap = (usage?.swap) || 0

  let modified = false
  let newSwap = 0;

  const dec = () => {
    if (modified) {
      --newSwap
    } else {
      modified = true
      newSwap = swap - 1
    }
    if (newSwap <= 0) {
      newSwap = 0
    }
  }

  const inc = () => {
    if (modified) {
      ++newSwap
    } else {
      modified = true
      newSwap = swap + 1
    }
    const free = parseInt(disk[2] / (1024 * 1024 * 1024) / 2)
    if (newSwap > free) {
      newSwap = free
    }
  }

  const setModify = () => {
    if (!modified) {
      modified = true
      newSwap = swap
    }
  }
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

      <div class="btn-label">Commands</div>
      <div class="buttons">
        <button on:click={restartGroundSeg} class="btn">Restart GroundSeg</button>
        {#if state == "pending"}
          <button on:click={updateLinux} class="btn">Update Linux</button>
        {:else}
          <div class="spacer"></div>
        {/if}
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
      <div class="row">
        <div class="label">Swap Memory</div>
        <div class="swap-wrapper">
          <div class="spacer"></div>
          <div class="btn-swap" on:click={dec}>-</div>
          <div class="custom" on:click={setModify}>
            {#if modified}
              <input style="width:calc(24px * {JSON.stringify(newSwap).length});" type="number" bind:value={newSwap} />
            {:else}
              <input style="width:calc(24px * {JSON.stringify(swap).length});" type="number" bind:value={swap} />
            {/if}
            <div class="details">GB</div>
          </div>
          <div on:click={inc} class="btn-swap">+</div>
          <div class="spacer"></div>
        </div>
        <div class="buttons">
          <button on:click={()=>modified = false} disabled={!modified || (swap == newSwap)} class="btn">Reset Change</button>
          <button on:click={()=>setSwap(newSwap)} disabled={(swap == newSwap) || !modified} class="btn">Modify Swap</button>
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
  .title {
    margin-bottom: 24px;
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
    border-radius: 8px;
    /*border: solid 4px var(--btn-secondary);*/
    border: solid 4px var(--btn-primary);
    height: 24px;
  }
  .ram-bg {
    position: absolute;
    left: 0;
    top: 0;
    background: var(--btn-primary);
    /*background: var(--btn-secondary);*/
    height: 100%;
    transition: width 500ms;
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
    display: flex;
    gap: 40px;
  }
  .btn {
    border-radius: 12px;
    flex: 1;
    background-color: var(--btn-secondary);
    color: var(--text-card-color);
    padding: 12px;
    font-family: var(--regular-font);
    font-size: 12px;
  }
  .btn:disabled {
    opacity: .6;
    pointer-events: none;
  }
  .btn-label {
    font-family: var(--regular-font);
    font-size: 12px;
    padding-left: 10px;
    margin-bottom: 12px;
  }
  .spacer {
    flex: 1;
  }
  .swap-wrapper {
    display: flex;
    align-items: end;
    gap: 20px;
    margin-bottom: 20px;
  }
  .custom {
    display: flex;
    align-items: center;
  }
  input[type=number] {
    min-width: 48px;
    font-family: var(--title-font);
    font-size: 32px;
    line-height: 32px;
    background: none;
    border:none;
  }
  input::-webkit-outer-spin-button,
  input::-webkit-inner-spin-button {
      -webkit-appearance: none;
        margin: 0;
  }
  input[type=number] {
      -moz-appearance: textfield;
  }
  input:focus {
    outline: none;
  }
  .details {
    font-family: var(--title-font);
    font-size: 32px;
    line-height: 32px;
    color: var(--text-color);
  }
  .btn-swap {
    user-select: none;
    width: 62px;
    height: 32px;
    padding-left: 2px;
    border-radius: 8px;
    background: var(--btn-primary);
    line-height: 26px;
    font-family: var(--title-font);
    font-size: 24px;
    text-align: center;
    color: var(--text-card-color);
  }
</style>
