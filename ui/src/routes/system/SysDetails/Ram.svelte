<script>
  import './theme.css'

  export let ram
  $: ramPercent = (ram[1]/ram[0] * 100).toFixed(2)
</script>

<div class="info-wrapper">
  <div class="info-title">RAM</div>
  <div class="info-text">
    {(ram[1] / (1024 * 1024 * 1024)).toFixed(1)} GB / {(ram[0] / (1024 * 1024 * 1024)).toFixed(1)} GB
  </div>
</div>
<div class="bar-bg">
  <div class="bar-fg" style="width:{ramPercent}%"></div>
</div>
