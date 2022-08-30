<script>
  import Logo from '/src/Components/Buttons/Logo.svelte'
  import SysInfo from '/src/Components/SysInfo.svelte'
  import Power from '/src/Components/Power.svelte'
  import Network from '/src/Components/Network.svelte'
  import MinIO from '/src/Components/MinIO.svelte'
  import Anchor from '/src/Components/Anchor.svelte'
  import Bitcoin from '/src/Components/Bitcoin.svelte'
  import Logs from '/src/Components/Logs.svelte'

</script>

<Logo t="System Settings" back=true />
<div class="container">
  <div class="panel">
    <Power />
    <SysInfo />
    <Network />
    <MinIO />
  </div>
  <div class="panel">
    <Anchor />
    <Bitcoin />
    <Logs />
  </div>
</div>

<style>
  .container {
    padding: 20px;
    width: 772px;
    max-width: calc(100vw - 40px);
    overflow: auto;
    display: flex;
    flex-wrap: wrap;
    gap: 12px;
  }
  .panel {
    display: flex;
    flex-direction: column;
    gap: 12px;
    width: 380px;
  }
</style>








