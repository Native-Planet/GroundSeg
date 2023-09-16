<script>
  import { structure,  startramRestart } from '$lib/stores/websocket'
  export let renew
  export let expiry
  export let registered
</script>

{#if registered}
  <div class="wrapper">
    <div class="title">STARTRAM</div>
    <div class="info-wrapper">
      <div class="info">
        <div class="item">Autorenew</div>
        <div class="item">{renew ? "Yes" : "No"}</div>
      </div>
      <div class="info">
        <div class="item">Expiration Date</div>
        <div class="item">{expiry}</div>
      </div>
    </div>
  </div>
{:else}
  <div>
    <div class="title">STARTRAM</div>
    <div class="info-box">Not Registered</div>
  </div>
{/if}

<style>
  .title {
    color: #000;
    leading-trim: both;
    text-edge: cap;
    font-family: var(--title-font);
    font-size: 32px;
    font-style: normal;
    font-weight: 700;
    line-height: normal;
    letter-spacing: -1.92px;
    margin-bottom: 32px;
  }
  .info-wrapper {
    display: flex;
    margin-bottom: 12px;
    gap: 56px;
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
    line-height: 48px; /* 200% */
    letter-spacing: -1.44px;
  }
</style>
