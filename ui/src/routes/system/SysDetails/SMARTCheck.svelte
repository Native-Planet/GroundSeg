<script>
  import './theme.css'
  export let smart = {};

</script>

  <div class="volume-info">
    <div class="info-wrapper">
      <div class="info-title">S.M.A.R.T Results</div>
      {#each Object.entries(smart) as [name,pass]}
        <div class="smart-wrapper">
          <div class="info-text">/dev/{name}</div>
          <div class="check-light" class:pass={pass}>{pass ? "pass" : "fail"}</div>
        </div>
      {/each}
    </div>
  </div>

<style>
  .info-wrapper {
    flex-direction: column;
  }
  .smart-wrapper {
    display: flex;
    gap: 32px;
    align-items: center;
  }
  .info-text {
    width: 140px;
  }
  .check-light {
    background: red;
    color: white;
    padding: 8px 0;
    width: 120px;
    margin: 4px;
    border-radius: 16px;
    text-align: center;
  }
  .pass {
    background: lime;
  }
</style>
