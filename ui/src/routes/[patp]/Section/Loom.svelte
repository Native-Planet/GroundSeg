<script>
  export let loomSize
  let curLoomSize = loomSize
  $: adjusted = Math.log2(curLoomSize * 1024 * 1024)
</script>

<div class="section">
  <div class="section-left">
    <div class="section-title">Loom Size</div>
    <div class="section-description">
      This function will refragment your ship’s memory capacity, optimizing its performance. We recommend scheduling these once a week
    </div>
  </div>
  <div class="section-right">
    <input type="range" min="28" max="33" step="1" class="range" bind:value={adjusted}>
    <div class="labels">
      <div class="label danger-loom">256M</div>
      <div class="label danger-loom">512M</div>
      <div class="label danger-loom">1G</div>
      <div class="label">2G</div>
      <div class="label">4G</div>
      <div class="label">8G</div>
    </div>
  </div>
</div>

<style>
  .section-right {
    display: flex;
    align-items: center;
    flex-direction: column;
    gap: 4px;
  }
  .range {
    -webkit-appearance: none;
    width: 240px;
    height: 11px;
    background: #ffffff4d;
    border-radius: 16px;
  }
  .danger-loom {
    color: #ffa500;
  }
  .range::-webkit-slider-thumb {
    -webkit-appearance: none;
    appearance: none;
    width: calc((240px / 6) - 8px);
    height: 16px;
    background: var(--btn-secondary);
    cursor: pointer;
    border-radius: 16px;
  }
  .range::-moz-range-thumb {
    width: calc(240px / 6);
    height: 16px;
    background: var(--btn-secondary);
    cursor: pointer;
    border-radius: 16px;
  }
  .labels {
    display: flex;
    justify-content: space-between;
    width: 240px;
    padding-bottom: 6px;
  }
  .label {
    width: calc(180px / 5);
    font-size: 11px;
  }
</style>
