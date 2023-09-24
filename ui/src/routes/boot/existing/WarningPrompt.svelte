<script>
  import { closeModal } from 'svelte-modals'
  import Modal from '$lib/Modal.svelte'

  import { warningDone } from './store'

  export let isOpen

  const kickstartUpload = () => {
    warningDone.set(true)
    closeModal()
  }
</script>
{#if isOpen}
  <Modal>
    <div class="wrapper">
      <h1>Is your pier offline?</h1>
      <p>Confirm that your pier is offline before importing your pier. It will be corrupted if it is online somewhere else.</p>
      <div class="buttons-wrapper">
        <button
          on:click={closeModal}
          >No, it is not
        </button>
        <button
          on:click={kickstartUpload}
          class="confirm"
          >Yes, it is offline
        </button>
      </div>
    </div>
  </Modal>
{/if}
<style>
  .wrapper {
    margin: 32px;
    font-family: var(--regular-font);
  }
  h1 {
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
  p {
    color: var(--Gray-400, #5C7060);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    letter-spacing: -1.44px;
    width: 500px;
  }
  .buttons-wrapper {
    display: flex;
    gap: 16px;
  }
  button {
    margin-top: 56px;
    background-color: var(--btn-secondary);
    border-radius: 16px;
    cursor: pointer;
    padding: 0 48px;
    height: 65px;
    color: #FFF;
    text-align: center;
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    line-height: 32px; /* 133.333% */
    letter-spacing: -1.44px;
  }
  .confirm {
    background: var(--NP_Green, #047710);
  }
</style>
