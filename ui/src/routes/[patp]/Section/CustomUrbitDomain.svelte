<script>
  // Style
  import "../theme.css"
  import { setUrbitDomain } from '$lib/stores/websocket'
  export let patp
  let domain = ""
</script>

<div>
  <div class="section-title-wrapper">
    <div class="section-title">Custom Urbit Domain</div>
    <div class="what">?</div>
  </div>
  <div class="wrapper">
    <input type="text" placeholder="ship.example.com" bind:value={domain} />
    <button disabled={domain.length < 1} class="save-button" on:click={()=>setUrbitDomain(patp, domain)}>Save</button>
  </div>
</div>

<style>
  .section-title-wrapper {
    display: flex; 
    align-items: center;
    gap: 16px;
  }
  .wrapper {
    display: flex;
    gap: 16px;
    margin: 16px 0;
  }
  input {
    flex: 1;
    align-items: center;
    border-radius: 16px;
    color: var(--Gray-200, #ABBAAE);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 32px;
    font-style: normal;
    font-weight: 300;
    line-height: normal;
    letter-spacing: -1.92px;
    padding: 14px;
    border: 2px solid var(--text-card-color);
    background: var(--bg-card);
  }
  .save-button {
    padding: 20px 48px;
    justify-content: center;
    align-items: center;
    gap: 8px;
    border-radius: 16px;
    background: #2C3A2E;
    color: var(--Gray-200, #ABBAAE);
    font-family: Inter;
    font-size: 32px;
    font-style: normal;
    font-weight: 300;
    line-height: 24px; /* 75% */
    letter-spacing: -1.92px;
    cursor: pointer;
  }
  .save-button:disabled {
    opacity: .6;
    pointer-events: none;
  }
  .what {
    width: 20px;
    height: 20px;
    text-align: center;
    border: 1px solid #FFF;
    border-radius: 50%;
    cursor: pointer;
    font-size: 16px;
  }
  .what:hover {
    opacity: .2;
  }
</style>
