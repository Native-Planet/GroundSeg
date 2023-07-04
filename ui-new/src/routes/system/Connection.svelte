<script>
  export let wide
</script>

<div class="container {wide ? "wide" : "slim"}">
  <div class="title">Connection</div>
  <div class="wifi">
    <div class="wifi-text">Wi-Fi</div>
    <div class="btn-wifi"></div>
  </div>
</div>

<style>
  .wide {
    width: 992px;
    max-width: 98vw;
    padding: 56px;
  }
  .slim {
    width: 100vw;
  }
  .wifi {
    margin-bottom: 32px;
    display: flex;
  }
  .wifi-text {
    font-size: 14px;
  }
</style>
