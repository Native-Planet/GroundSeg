<script>
  import { onMount } from 'svelte'
  import Logo from '/src/Components/Buttons/Logo.svelte'
  import Boot from '/src/Components/Buttons/Boot.svelte'

  let piers

  onMount(async () => {
    fetch("http://localhost:5000/")
      .then(res => res.json())
      .then(data => piers = data)
  })

</script>

<div class="home">
  <Logo />
  <div class="list">
    {#if piers}
      {#each piers as p}
        <div class="pier">
          <div class="sigil">
          </div>
          <div class="info">
            <div class="patp">~{p.name}</div>
            <div class="status">
              {p.running ? "Running" : "Stopped"}
            </div>
          </div>
          <img class="gear" src="/pier_settings.png" alt="gear" />
        </div>
      {/each}
    {/if}
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
