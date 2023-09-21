<script>
  // Style
  import "../theme.css"
  import Clipboard from 'clipboard'
  import { createEventDispatcher } from 'svelte'
  export let url = "#"
  export let lusCode = ""
  const dispatch = createEventDispatcher()
  let copied = false

  let copy = new Clipboard('#lus-code');
    copy.on("success", ()=> {
      copied = true;
      setTimeout(()=> copied = false, 1000)
    })
</script>

<div>
  <div class="section-title">Custom Urbit Domain</div>
  <div class="wrapper">
    <input type="text" placeholder="ship.example.com" />
    <button class="save-button">Save</button>
  </div>
  <div class="wrapper">
    <button id="lus-code" class="btn" data-clipboard-text={lusCode}>
      <img
        src="/clipboard.svg"
        width="24px"
        height="24px" />
      {#if copied}
        Copied!
      {:else}
        Access Key
      {/if}
    </button>
    <a href={url} target="_blank" class="btn">
      URL ↗
    </a>
  </div>
</div>

<style>
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
  }
  .btn {
    color: #161D17;
    text-align: right;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    line-height: 24px; /* 100% */
    letter-spacing: -1.44px;
    padding: 16px;
    display: flex;
    align-items: center;
    gap: 8px;
    border-radius: 12px;
    background: var(--NP_White, #F8F8F6);
  }
</style>
