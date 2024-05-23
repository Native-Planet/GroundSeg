<script>
  import Fa from 'svelte-fa'
  import { faCircleNotch, faTrashCan } from '@fortawesome/free-solid-svg-icons';

  import { onMount } from 'svelte'
  import { structure } from '$lib/stores/data'
  import { startramGetServices, deleteStartramService } from '$lib/stores/websocket'

  $: services = $structure?.profile?.startram?.info?.startramServices || []
  $: urbitJSONKeys = Object.keys($structure?.urbits)
  let serviceNames = ["minio", "minio-bucket", "minio-console", "urbit-ames", "urbit-web"]
  let loading = [] // patps

  const handleDeleteServices = patp => {
    // for each service in an array of string called services do:
    serviceNames.forEach(service => {
      if (!loading.includes(patp)) {
        loading.push(patp)
      }
      deleteStartramService(patp, service)
    })
  }

  onMount(()=>startramGetServices())
</script>

{#if services.length < 1}
  <div class="no-ships-text">No startram services registered</div>
{:else}
  <div class="ships">
    <!-- registered ships -->
    {#each services as patp}
      {#if !urbitJSONKeys.includes(patp)}
      <div class="option">
        <div class="patp">{patp}</div>
        {#if loading.includes(patp)}
          <div class="deleting">
            <Fa icon={faCircleNotch} size="1x" />
          </div>
        {:else}
          <button on:click={()=>handleDeleteServices(patp)} class="delete-service">
            <Fa icon={faTrashCan} size="1x"/>
          </button>
        {/if}
      </div>
      {/if}
    {/each}
  </div>
{/if}

<style>
  .no-ships-text {
    font-size: 20px;
    text-align: left;
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-style: normal;
    font-weight: 300;
    letter-spacing: -1.44px;
  }
  .ships {
    display: flex; 
    flex-direction: column;
    gap: 32px;
  }
  .option {
    display: flex;
    max-width: 480px;
    gap: 16px; 
  }
  .patp {
    flex: 1;
    font-size: 24px;
    text-align: left;
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-style: normal;
    letter-spacing: -1.44px;
  }
  .delete-service {
    cursor: pointer;
    background: var(--btn-secondary);
    color: white;
    padding: 8px;
    border-radius: 8px;
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-style: normal;
    letter-spacing: -1.44px;
    font-size: 18px;
    text-align: center;
  }
  .deleting {
    color: var(--btn-secondary);
    padding: 8px;
    border-radius: 8px;
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-style: normal;
    letter-spacing: -1.44px;
    font-size: 18px;
    text-align: center;
    animation: rotate 1s linear infinite;
  }

  @keyframes rotate {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}
</style>
