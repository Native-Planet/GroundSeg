<script>
  import { structure,  startramRestart } from '$lib/stores/websocket'
  export let renew
  export let expiry
  export let registered
</script>

{#if registered}
  <div class="wrapper">
    <div class="prof-title">STARTRAM</div>
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
</style>
