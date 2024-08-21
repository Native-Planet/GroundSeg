<script>
  import Fa from 'svelte-fa'
  import { faCaretUp, faCaretDown } from '@fortawesome/free-solid-svg-icons';
  // Modals
  import { openModal } from 'svelte-modals'
  import EndpointModal from './EndpointModal.svelte'
  import RegisterModal from './RegisterModal.svelte'

  import StarTramReminder from './StarTramReminder.svelte'
  import StarTramServices from './StarTramServices.svelte'

  import { structure } from '$lib/stores/data'

  $: info = ($structure?.profile?.startram?.info) || {}
  $: endpoint = (info?.endpoint) || ""
  $: registered = (info?.registered) || false
  $: region = (info?.region) || ""
  $: regionFormatted = capitalizeFirstLetter(region.replace(/-/g, ' '))

  const capitalizeFirstLetter = str => str.replace(/\w\S*/g, txt => txt.charAt(0).toUpperCase() + txt.substr(1).toLowerCase());
  let viewReminder = false
  let viewServices = false
</script>

<div class="wrapper">
  <div class="item">
    <div class="label">Endpoint</div>
    <div class="data">
      <div class="data-text">{endpoint}</div>
      <button
      on:click={()=>openModal(EndpointModal)}
      >Edit</button>
    </div>
  </div>

  {#if registered && (region.length > 0)}
    <div class="item">
      <div class="label">Region</div>
      <div class="data">
        <div class="data-text">{regionFormatted}</div>
      <button
        on:click={()=>openModal(RegisterModal,{"regionMode":true})}
      >Edit</button>
      </div>
    </div>
  {/if}

  {#if registered}
    <div class="item">
      <div class="label">Remote Backup</div>
      <div class="data">
        <div class="data-text">Your next backup is at "time here"</div>
        <button>Backup Now</button>
      </div>
    </div>
  {/if}

  <div class="item">
    <div class="label"
         on:click={()=>viewReminder=!viewReminder}>
      StarTram expiration reminder on Urbit
      <Fa icon={viewReminder ? faCaretUp : faCaretDown} size="1x"/>
    </div>
    {#if viewReminder}
      <StarTramReminder />
    {/if}
  </div>

  <div class="item">
    <div class="label"
       on:click={()=>viewServices=!viewServices}>
       Orphaned services
      <Fa icon={viewServices ? faCaretUp : faCaretDown} size="1x"/>
    </div>
    {#if viewServices}
      <StarTramServices />
    {/if}
  </div>

</div>

<style>
  .wrapper {
    display: flex;
    flex-direction: column;
    margin-top: 64px;
    gap: 32px;
  }
  .label {
    cursor: pointer;
    font-size: 17px;
    margin-bottom: 21px;
    display: flex;
    gap: 20px;
    align-items: center;
  }
  .data {
    display: flex;
    gap: 21px;
    height: 65px;
    align-items: center;
  }
  .data-text {
    flex: 1; 
    background: var(--bg-modal);
    border-radius: 16px;
    padding-left: 24px;

    color: var(--NP_Black, #313933);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    line-height: 65px;
    letter-spacing: -1.44px;
  }
  button {
    font-size: 24px;
    line-height: 65px;
    background: var(--btn-secondary);
    border-radius: 16px;
    color: #FFF;
    text-align: center;
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    letter-spacing: -1.44px;
    padding: 0 48px;
  }
</style>
