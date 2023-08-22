<script>
  import './theme.css'

  export let disk
  $: diskPercent = (disk[1]/disk[0] * 100).toFixed(2)
</script>

<div class="info-wrapper">
  <div class="info-title">Storage</div>
  <div class="info-text">
    {(disk[1] / (1024 * 1024 * 1024)).toFixed(1)} GB / {(disk[0] / (1024 * 1024 * 1024)).toFixed(1)} GB
  </div>
</div>
<div class="bar-bg">
  <div class="bar-fg" style="width:{diskPercent}%"></div>
</div>
