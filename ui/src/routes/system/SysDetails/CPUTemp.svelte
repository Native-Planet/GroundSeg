<script>
  import './theme.css'
  export let cpuTemp
</script>

<div class="info-wrapper">
  <div class="info-title">CPU Temperature</div>
  <div class="info-text">{cpuTemp}%</div>
</div>
<div class="bar-bg">
  <div class="bar-fg" style="width:{cpuTemp}%"></div>
</div>
