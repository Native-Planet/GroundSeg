<script>
  // Style
  import "../theme.css"

  import Clipboard from 'clipboard'
  import { createEventDispatcher } from 'svelte'

  export let minioUrl
  export let minioPwd

  let copied = false

  const dispatch = createEventDispatcher()

  let copy = new Clipboard('#copy');
    copy.on("success", ()=> {
      copied = true;
      setTimeout(()=> copied = false, 1000)
    })

</script>

<div>
  <div class="section-title">MinIO</div>
  <div class="wrapper">
    <input type="text" placeholder="minio.example.com" />
    <button class="save-button">Save</button>
  </div>
  <div class="wrapper">
    <button id="copy" class="btn" data-clipboard-text={minioPwd}>
      <img
        src="/clipboard.svg"
        width="24px"
        height="24px" />
      {#if copied}
        Copied!
      {:else}
        Copy MinIO Password
      {/if}
    </button>
    <a href={minioUrl} target="_blank" class="btn">
      Settings
    </a>
    <button class="btn">
      Disconnect
    </button>
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
