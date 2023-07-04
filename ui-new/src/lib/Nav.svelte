<script>
  let wide = true
</script>

<div class="wrapper {wide ? "wide" : "slim"}">
  <div class="version">V2.0.0</div>
  <div class="app">GROUNDSEG</div>
  <div class="nav">
    <div class="ships">
      <div class="text">SHIPS</div>
      <div class="add"></div>
    </div>
    <div class="option">PROFILE</div>
    <div class="option">SYSTEM</div>
  </div>
</div>

<style>
  .wide {
    width: calc((288px * 3) + (120px * 2));
    max-width: 98vw;
  }
  .slim {
    width: 100vw;
  }
  .wrapper {
    color: var(--text-color);
    margin: auto;
    margin-top: 10px;
  }
  .version {
    font-size: 12px;
  }
  .app {
    font-size: 16px;
    margin-bottom: 16px;
  }
  .nav {
    display: flex;
    margin-bottom: 10px;
    gap: 24px;
  }
  .ships {
    flex: 1;
    display: flex;
    align-items: center;
    gap: 10px;
  }
  .text {
    font-size: 24px;
  }
  .add {
    background: #06A315;
    width: 24px;
    height: 24px;
    border-radius: 100%;
  }
  .option {
    font-size: 24px;
  }
</style>
