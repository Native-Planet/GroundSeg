<script>
  export let info
</script>

<div class="sys">
  <div class="sys-title">System Information</div>
  {#if info}
    <div class="hw">
      <div class="word">RAM</div>
      <div class="data">{info.ram}%</div>
    </div>
    <div class="hw">
      <div class="word">CPU Temperature</div>
      <div class="data">{info.temp} &deg C</div>
    </div>
    <div class="hw">
      <div class="word">Storage</div>
      <div class="data">
        <span>{(info.disk[1] / (1000 * 1000 * 1000)).toFixed(1)}</span> 
        <span>GB / </span>
        <span>
          {(info.disk[0] / (1000 * 1000 * 1000)).toFixed(1)} GB
        </span>
      </div>
    </div>
    {:else}
    <div class="hw">
      <div class="word">RAM</div>
      <div class="data blurred"></div>
    </div>
    <div class="hw">
      <div class="word">CPU Temperature</div>
      <div class="data blurred"></div>
    </div>
    <div class="hw">
      <div class="word">Storage</div>
      <div class="data blurred-long"></div>
    </div>
  {/if}
</div>

<style>
  @keyframes breathe {
    0% {opacity: .6}
    50% {opacity: 0}
    100% {opacity: .6}
  }

  .sys {
    display: flex;
    flex-direction: column;
    background: #0000006d;
    width: 300px;
    padding: 40px;
    border-radius: 15px;
    font-size: 18px;
    gap: 12px;
  }
  .sys-title {
    font-size: 18px;
    padding-bottom: 8px;
  }

  .hw {
    display: flex;
    font-size: 14px;
  }
  .word {
    flex: 1;
  }

  .blurred {
    width: 60px;
    animation: breathe 2s infinite;
    background: #ffffff4d;
    filter: blur(6px);
    border-radius: 8px;
  }
  .blurred-long {
    width: 160px;
    animation: breathe 2s infinite;
    background: #ffffff4d;
    filter: blur(6px);
    border-radius: 8px;
  }
</style>
