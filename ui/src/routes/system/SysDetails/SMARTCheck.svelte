<script>
  import './theme.css'
  export let smart = {};

</script>

  <div class="volume-info">
    <div class="info-wrapper">
      <div class="info-title">Hard Drive Health</div>
      <div class="smart">
      {#each Object.entries(smart) as [name,pass]}
        <div class="smart-wrapper">
          <div class="info-text">/dev/{name}</div>
          <div class="check-light" class:pass={pass}>{pass ? "pass" : "fail"}</div>
        </div>
      {/each}
      </div>
    </div>
  </div>

<style>
  .info-wrapper {
    flex-direction: column;
  }
  .smart {
    margin-top: 24px;
  }
  .smart-wrapper {
    display: flex;
    gap: 32px;
    align-items: center;
  }
  .info-text {
    width: 100px;
    font-size: 18px;
  }
  .check-light {
    background: red;
    color: white;
    padding: 4px 0;
    width: 80px;
    margin: 4px;
    border-radius: 8px;
    text-align: center;
    font-size: 18px;
  }
  .pass {
    background: lime;
  }
</style>
