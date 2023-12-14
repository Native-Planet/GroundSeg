<script>
  import Slider from './slider.svelte'
  export let patp
  export let loomSize
</script>

<div class="section">
  <div class="section-left">
    <div class="section-title">Loom Size</div>
    <div class="section-description">
      This function will modify your ship's maximum allowed RAM usage. We recommend 2048 MB to 8192 MB depending on usage
    </div>
  </div>
  <div class="section-right">
    <div class="slider">
      <div class="spacer"></div>
      <Slider {patp} {loomSize}/>
    </div>
  </div>
</div>

<style>
  .slider {
    display: flex;
  }
  .spacer {
    flex: 1;
  }
</style>
