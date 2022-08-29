  <div class="sys">
    <div class="sys-title">System Information</div>
    <div class="hw">
      <div class="word">RAM</div>
      <div class="data">12.5%</div>
    </div>
    <div class="hw">
      <div class="word">CPU Temperature</div>
      <div class="data">36 &deg C</div>
    </div>
    <div class="hw">
      <div class="word">Storage</div>
      <div class="data">123 GB / 133 GB</div>
    </div>
  </div>

<style>
  .sys {
    display: flex;
    flex-direction: column;
    background: #0000006d;
    width: 300px;
    max-width: calc(100% - 80px);
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
</style>
