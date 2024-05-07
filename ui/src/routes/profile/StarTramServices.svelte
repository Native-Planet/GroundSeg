<script>
  import Fa from 'svelte-fa'
  import { faCircleExclamation } from '@fortawesome/free-solid-svg-icons';

  import { onMount } from 'svelte'
  import { structure } from '$lib/stores/data'
  import { startramGetServices, deleteStartramService } from '$lib/stores/websocket'
  $: services = $structure?.profile?.startram?.info?.startramServices || {}
  $: urbitKeys = Object.keys(services)
  $: urbitJSONKeys = Object.keys($structure?.urbits)
  let serviceNames = ["minio", "minio-bucket", "minio-console", "urbit-ames", "urbit-web"]
  /*
  $: selectAllCheck = selectSwitcher(urbits)
   
  const selectSwitcher = u => {
    let reminderStatus = [];
    for (let patp in u) {
      if (!u[patp].info.startramReminder) {
        return false
      }
    }
    return true
  }
*/

  const handleDeleteServices = patp => {
    // for each service in an array of string called services do:
    serviceNames.forEach(service => {
      deleteStartramService(patp, service)
    })
  }

  onMount(()=>startramGetServices())
</script>
{#if urbitKeys.length < 1}
  <div class="no-ships-text">No startram services registered</div>
{:else}
  <div class="ships">
    {#each urbitKeys as patp}
      <div class="option">
        <div class="patp">{patp}</div>
        {#if !urbitJSONKeys.includes(patp)}
          <div class="patp-skull">
            <Fa icon={faCircleExclamation} size="1x" color="black"/>
            <div class="ship-not-found">ship not found on device!</div>
          </div>
        {/if}
        <div class="service-info">
          {#each serviceNames as svc}
            <div class="service-option">
              <div class="service-name">{svc}</div>
              <div
                class="service-status"
                on:click={()=>deleteStartramService(patp,svc)}>
                {services[patp][svc].status}
              </div>
            </div>
          {/each}
        </div>
        <button on:click={()=>handleDeleteServices(patp)} class="delete-service">Delete StarTram Services</button>
      </div>
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
    flex-direction: column;
    gap: 16px; 
    max-width: 400px;
    flex: 1;
  }
  .patp {
    font-size: 24px;
    text-align: left;
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-style: normal;
    letter-spacing: -1.44px;
  }
  .patp-skull {
    display: flex;
    gap: 8px;
    background: orange;
    padding: 8px;
    align-items: center;
    height: 16px;
    width: 200px;
  }
  .ship-not-found {
    font-size: 16px;
    text-align: left;
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-style: normal;
    letter-spacing: -1.44px;
  }
  .service-info {
    display: flex;
    flex-direction: column;
  }
  .service-option {
    display: flex;
  }
  .service-name {
    flex: 1;
  }
  .service-status {
    flex: 1;
  }
  .delete-service {
    cursor: pointer;
    background: black;
    color: white;
    padding: 4px;
    width: 220px;
    border-radius: 8px;
    text-align: left;
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-style: normal;
    letter-spacing: -1.44px;
    font-size: 18px;
    text-align: center;
  }
</style>
