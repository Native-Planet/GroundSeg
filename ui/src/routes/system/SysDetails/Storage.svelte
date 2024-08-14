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

  <div class="volume-info">
    <div class="info-wrapper">
      <div class="info-title">Filesystem Usage</div>
      {#each Object.entries(disk) as [volume, [used, total]]}
        <div class="disk">
          <div class="disk-title">{decodeLabel(volume)}</div>
          <div class="info-text">
            {formatBytes(used)} used / {formatBytes(total)} total
          </div>
        </div>
        <div class="bar-bg">
          <div class="bar-fg" style="width:{(100 * used / total).toFixed(2)}%"></div>
        </div>
      {/each}
    </div>
  </div>

<style>
  .info-wrapper {
    flex-direction: column;
  }
  .disk {
    display: flex;
    align-items: end;
    width: 100%;
    padding-left: 8px;
  }
  .disk-title {
    flex: 1;
    color: var(--NP_Black, #161D17);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 16px;
    font-style: normal;
    font-weight: 300;
    letter-spacing: -0.96px;
    margin-top: 16px;
  }
  button {
    margin-top: 32px;
    border-radius: 16px;
    background: var(--Gray-400, #5C7060);
    color: #FFF;

    text-align: center;
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    height: 64px;
    font-style: normal;
    font-weight: 300;
    line-height: 32px; /* 133.333% */
    letter-spacing: -1.44px;
    padding: 0 48px;
  }
  button:disabled {
    opacity: 0.6;
    pointer-events: none;
  }
</style>
