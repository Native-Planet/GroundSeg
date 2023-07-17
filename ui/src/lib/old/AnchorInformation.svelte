<script>
  export let region
  export let regions
  export let ongoing
  export let lease
</script>

<div class="info-wrapper">
  {#if lease != null}
    {#if region != null}
      <div class="info">
        <div class="info-header">
          Active Region
        </div>
        <div class="info-content">
          {regions.find(obj => obj.name == region)?.desc}
        </div>
      </div>
    {/if}
    <div class="info">
      <div class="info-header">
        Expiry Date
      </div>
      <div class="info-content">{lease}</div>
    </div>
    <div class="info">
      <div class="info-header">
        Auto Renew
      </div>
      <div class="info-content">
        {ongoing ? "Yes" : "No"}
      </div>
    </div>
  {/if}
</div>

<style>
  .info-wrapper {
    display: flex;
    flex-direction: column;
    gap: 6px;
    margin-top: 16px;
    max-width: 180px;
  }
  .info {
    display: flex;
    text-align: center;
    border: solid 1px white;
    padding: 4px 12px;
    gap: 10px;
  }
  .info-header {
    flex: 1;
    font-size: 10px;
    line-height: 16px;
    text-align: right;
  }
  .info-content {
    flex: 1;
    font-size: 11px;
    line-height: 16px;
    text-align: left;
  }

</style>
