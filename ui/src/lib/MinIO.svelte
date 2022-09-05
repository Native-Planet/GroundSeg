<script>
  let off = false
  const toggleMinIO = () => off = !off
</script>

  <div class="minio">
    <div class="title-wrapper">
      <div class="title">MinIO</div>
      <div on:click={toggleMinIO} class="switch-wrapper">
        <div class="switch {off ? "off" : "on"}"></div>
      </div>
    </div>
  </div>

<style>
  .minio {
    background: #0000006d;
    width: 300px;
    padding: 20px 40px 20px 40px;
    border-radius: 15px;
    font-size: 18px;
  }
  .title-wrapper {
    display: flex;
    align-items: center;
  }
  .title {
    font-size: 18px;
    flex: 1;
  }
  .switch-wrapper {
    border-radius: 8px;
    width: 32px;
    height: 12px;
    background: #ffffff4d;
    padding: 2px;
  }
  .switch {
    height: 100%;
    width: 19px;
    border-radius: 6px;
    margin-top: auto;
  }
  .on {
    background: #008eff;
    float: right;
  }
  .off {
    background: #000;
    float: left;
    opacity: .2;
  }
</style>
