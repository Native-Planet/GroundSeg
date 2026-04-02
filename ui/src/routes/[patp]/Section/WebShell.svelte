<script>
  import { openModal } from 'svelte-modals'
  import WebShellModal from '../WebShellModal.svelte'

  import "../theme.css"

  export let patp

  const handleOpen = () => {
    openModal(WebShellModal, { patp })
  }
</script>

<div class="section">
  <div class="section-left">
    <div class="section-title">Web Shell</div>
    <div class="section-description">Attach to this ship's tmux session with <code>tmux a</code>.</div>
  </div>
  <div class="section-right">
    <button class="open-button" on:click={handleOpen}>Open Shell</button>
  </div>
</div>

<style>
  code {
    font-family: 'Source Code Pro', monospace;
    font-size: 90%;
  }
  .open-button {
    border-radius: 16px;
    cursor: pointer;
    padding: 18px 32px;
    color: var(--Gray-200, #ABBAAE);
    background: #2C3A2E;
    font-family: Inter;
    font-size: 20px;
    font-weight: 300;
    letter-spacing: -1.2px;
  }
  .open-button:hover {
    background: #39493c;
  }
</style>
