<script>
  import nplogo from '/src/routes/nplogo.png'
  import Fa from 'svelte-fa/src/fa.svelte'
  import { faWrench, faHouse } from '@fortawesome/free-solid-svg-icons'

  export let page
</script>
  <div class="header">
    {#if page === 'home'}
    <a href="/">
      <img class="nplogo" alt="NativePlanet Logo" src={nplogo} />
    </a>
    <div class="gap"></div>
      <a href="/boot/existing" class="boot">Boot an existing Urbit ID</a>
      <a href="/boot/new" class="boot">Boot a new Urbit ID</a>
      <a href="/settings">
        <Fa icon={faWrench} size="1.5x"/>
      </a>
    {:else}
    <a href="/">
        <Fa icon={faHouse} size="1.5x"/>
    </a>
    <div class="gap"></div>
    <div class="page">{page}</div>
    {/if}
  </div>
<style>
  .header {
    background-color: rgba(19,19,19,0.6);
    padding: 1em 2em 1em 2em;
    border-radius: 12px;
    display: flex;
    padding-bottom: 1em;
    box-shadow: 0 0 12px #fff;
    gap: 1em;
    align-items: center;
  }
  .gap {
    flex: 1;
  }
  .nplogo {
    height: 32px;
  }
  .boot {
    border: solid 1px #fff;
    padding: .8em;
    border-radius: 8px;
    font-size: .8em;
  }
  .boot:hover {
    color: white;
    opacity: .8;
  }
  .page {
    font-size: 1.2em;
    font-weight: 600;
  }
</style>
