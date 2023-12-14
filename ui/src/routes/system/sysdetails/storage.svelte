<script>
  import './theme.css'

  export let disk = {};

  function formatBytes(bytes, decimals = 2) {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const dm = decimals < 0 ? 0 : decimals;
    const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i];
  }

  function decodeLabel(label) {
    try {
      return decodeURIComponent(label);
    } catch (e) {
      return label;
    }
  }
</script>

{#each Object.entries(disk) as [volume, [used, total]]}
  <div class="volume-info">
    <div class="info-wrapper">
      <div class="info-title">{decodeLabel(volume)}</div>
      <div class="info-text">
        {formatBytes(used)} used / {formatBytes(total)} total
      </div>
    </div>
    <div class="bar-bg">
      <div class="bar-fg" style="width:{(100 * used / total).toFixed(2)}%"></div>
    </div>
  </div>
{/each}