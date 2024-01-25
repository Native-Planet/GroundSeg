<script>
  import { startramRestart } from '$lib/stores/websocket'
  import { startramMaxReminderDays, structure, daysUntilDate } from '$lib/stores/data'
  import Fa from 'svelte-fa'
  import { faTriangleExclamation } from '@fortawesome/free-solid-svg-icons';
  import { openModal } from 'svelte-modals'
  import StarTramReminderModal from '$lib/StarTramReminderModal.svelte'

  export let renew
  export let expiry
  export let registered

  $: daysLeft = daysUntilDate(expiry)

</script>

{#if registered}
  <div class="wrapper">
    <div class="prof-title" id="startram">STARTRAM</div>
    <div class="info-wrapper">
      <div class="info">
        <div class="item bold">Plan</div>
        <div class="item">{!renew ? "Subscription" : "Non-Recurring"}</div>
      </div>
      <div class="info">
        <div class="item bold">{!renew ? "Renewal Date" : "Expiration Date"}</div>
        <div class="item">{expiry}</div>
      </div>
    </div>
    {#if daysLeft <= $startramMaxReminderDays}
      <div class="warning">
        <Fa color="orange" icon={faTriangleExclamation} size="2x"/>
        <div class="warning-text">Your StarTram registration will expire in less than {daysLeft} day{daysLeft > 1 ? "s" : ""}.</div>
        <div class="renew-link" on:click={()=>openModal(StarTramReminderModal,{})}>convert plan</div>
      </div>
    {/if}
  </div>
{:else}
  <div>
    <div class="prof-title">STARTRAM</div>
    <div class="info-box">Not Registered</div>
  </div>
{/if}

<style>
  .prof-title {
    margin-bottom: 32px;
  }
  .info-wrapper {
    display: flex;
    margin-bottom: 12px;
    gap: calc(56px * 1.5);
  }
  .info {
    display: flex;
    flex-direction: column;
  }
  .info-box {
    border: solid 1px var(--text-color);
    padding: 16px;
    border-radius: 16px;
    opacity: .6;
    /* text */
    text-align: center;
    font-size: 15px;
    line-height: 32px;
  }
  .item {
    color: #000;
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    margin-bottom: 12px;
    letter-spacing: -1.44px;
  }
  .bold {
    font-weight: 500;
  }
  .warning {
    border: solid orange 2px;
    border-radius: 16px;
    padding: 20px;
    display: flex;
    flex-direction: column;
    align-items: center;
    color: var(--text-color);;
  }
  .warning-text {
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 18px;
    font-style: normal;
    font-weight: 300;
    margin-top: 16px;
    letter-spacing: -1.44px;
  }
  .renew-link {
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 20px;
    text-decoration: underline;
    font-weight: 300;
    margin: 8px;
    letter-spacing: -1.44px;
    color: var(--text-color);;
    cursor: pointer;
  }
</style>
