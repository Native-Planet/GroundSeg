<script>
  import { onMount } from 'svelte'
  import { structure } from '$lib/stores/data'
  import { startramGetServices, deleteStartramService } from '$lib/stores/websocket'
  $: services = $structure?.profile?.startram?.info?.startramServices || {}
  $: urbitKeys = Object.keys(services)
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

  onMount(()=>startramGetServices())
</script>
{#if urbitKeys.length < 1}
  <div class="no-ships-text">No startram services registered</div>
{:else}
  <div class="ships">
    <div class="option">
      <div class="patp">Ships</div>
      {#each serviceNames as svc}
        <div class="check-label">
          {svc}
        </div>
      {/each}
    </div>
    {#each urbitKeys as patp}
      <div class="option">
        <div class="patp">{patp}</div>
        {#each serviceNames as svc}
          <div class="checkbox" on:click={()=>deleteStartramService(patp,svc)}>
            {#if services[patp][svc].status == "ok"}
              <img class="checkmark" src="/checkmark.svg" alt="checkmark"  />
            {:else if services[patp][svc].status == "creating"}
              creating...
            {/if}
          </div>
        {/each}
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
  .spacer {
    flex: 1;
  }
  .option {
    display: flex;
    gap: 16px; 
    width: 600px;
    flex: 1;
  }
  .checkbox {
    width: 24px;
    height: 24px;
    border: solid 2px var(--text-color);
    border-radius: 8px;
    background: var(--bg-modal);
    cursor: pointer;
  }
  .patp {
    cursor: pointer;
    font-size: 24px;
    text-align: left;
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-style: normal;
    letter-spacing: -1.44px;
  }
  .checkmark {
    height: 18px;
    width: 18px;
    padding: 3px;
  }
</style>
