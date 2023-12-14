<script>
  import './theme.css'
  export let cpu
</script>

<div class="info-wrapper">
  <div class="info-title">CPU Load</div>
  <div class="info-text">{cpu}%</div>
</div>
<div class="bar-bg">
  <div class="bar-fg" style="width:{cpu}%"></div>
</div>
