<script>
  import Slider from './Slider.svelte'
  export let loomSize
</script>

<div class="section">
  <div class="section-left">
    <div class="section-title">Loom Size</div>
    <div class="section-description">
      This function will refragment your ship’s memory capacity, optimizing its performance. We recommend scheduling these once a week
    </div>
  </div>
  <div class="section-right">
    <div class="slider">
      <div class="spacer"></div>
      <Slider />
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
