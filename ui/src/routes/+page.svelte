<script>
  import Logo from '/src/Components/Buttons/Logo.svelte'
  import Boot from '/src/Components/Buttons/Boot.svelte'
  let piers = [
    'dister-dozzod-donpub-datdux',
    'tabtes-sorlec-mopfel-winrux',
    'famwyl-lavlyr-mopfel-winrux'
  ]
</script>

<div class="home">
  <Logo />
  <div class="list">
    {#each piers as p}
      <div class="pier">
        <div class="sigil">
        </div>
        <div class="info">
          <div class="patp">~{p}</div>
          <div class="status">Stopped</div>
        </div>
        <img class="gear" src="/pier_settings.png" alt="gear" />
      </div>
    {/each}
  </div>
  <Boot />
</div>

<style>
  .home {
    width: 500px;
    max-width: 80vw;
  }
  .list {
    margin: 24px 0 28px 0;
    display: flex;
    flex-direction: column;
  }
  .pier {
    padding: 6px 20px 6px 20px;
    display: flex;
    align-items: center;
    flex-wrap: wrap;
  }
  .pier:hover {
    background: #00000080;
  }
  .sigil {
    width: 60px;
    height: 60px;
    background: salmon;
    border-radius: 8px;
  }
  .info {
    display: flex;
    flex-direction: column;
    padding-left: 12px;
    flex: auto;
  }
  .patp {
    font-weight: 600;
    color: inherit;
  }
  .status {
    opacity: .8;
    font-weight: 400;
    font-size: .8em;

  }
  img, .gear {
    width: 28px;
    height: 28px;
  }
</style>
