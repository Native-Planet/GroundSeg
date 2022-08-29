<script>
  import Logo from '/src/Components/Buttons/Logo.svelte'
  import SysInfo from '/src/Components/SysInfo.svelte'
  import Power from '/src/Components/Power.svelte'
  import Network from '/src/Components/Network.svelte'
  import Anchor from '/src/Components/Anchor.svelte'
  import Bitcoin from '/src/Components/Bitcoin.svelte'
  import Logs from '/src/Components/Logs.svelte'

</script>

<div class="head">
  <Logo />
  <div class="title">Settings</div>
</div>
<div class="container">
  <div class="left-panel">
    <Power />
    <SysInfo />
    <Network />
  </div>
  <div class="right-panel">
    <Anchor />
    <Bitcoin />
    <Logs />
  </div>
</div>

<style>
  .head {
    display: flex;
    align-items: end;
  }
  .container {
    padding: 20px;
    width: 800px;
    max-width: calc(100vw - 40px);
    display: flex;
    flex-wrap: wrap;
    gap: 12px;
  }
  .left-panel {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }
  .right-panel {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }
  .title {
    font-size: 24px;
    padding: 0 0 4px 16px;
  }
</style>








