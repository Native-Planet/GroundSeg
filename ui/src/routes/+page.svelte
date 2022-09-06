<script>
  import { onMount } from 'svelte'
  import { api, piers } from '$lib/api'
  import { home } from '$lib/components'
  import Fa from 'svelte-fa'
  import { faGear } from '@fortawesome/free-solid-svg-icons/index.es'


  onMount(async () => {
    fetch(api).then(r => r.json()).then(d => piers.set(d))
  })

</script>

<div class="home">

  <svelte:component this={home.logo} />

  {#if $piers == null}

    <div class="loading"></div>

    {:else if Array.isArray($piers)}

      {#if $piers.length == 0}

        <div class="gap"></div> 

      {:else if $piers.length > 0}

        <div class="list">
          {#each $piers as p}
            <div class="pier">
              <svelte:component this={home.sigil} patp={p.name} size="60px" rad="8px" />
              <a class="info"
                href={p.running ? p.url : ""}
                target={p.running ? "_blank" : ""}>
                <div class="patp">{p.name}</div>
                <div class="status">{p.running ? "Running" : "Stopped"}</div>
              </a>
              <a href={p.name}>
                <Fa icon={faGear} size="1.2x" />
              </a>
            </div>
          {/each}
        </div>

      {/if}

    {/if}

  <svelte:component this={home.boot} />

</div>

<style>
  @keyframes breathe {
    0% {opacity: .6}
    50% {opacity: 0}
    100% {opacity: .6}
  }
  .home {
    width: 500px;
    max-width: 80vw;
    max-width: 100vw;
  }
  .gap {
    padding-bottom: 40px; 
  }
  .list {
    margin-bottom: 28px;
    margin-top: 8px;
    display: flex;
    flex-direction: column;
    max-height: 284px;
    overflow: auto;
    -ms-overflow-style: none;
    scrollbar-width: none;
  }

  .list::-webkit-scrollbar {
    display: none;
  }

  .loading {
    height: 80px;
    animation: breathe 2s infinite;
    background: #ffffff4d;
    filter: blur(10px);
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
</style>
