<script>
  import { structure, submitReport } from '$lib/stores/websocket'
  import { closeModal } from 'svelte-modals'
  import Modal from '$lib/Modal.svelte'
  export let isOpen
  let contact = ''
  let description = ''
  $: urbits = ($structure?.urbits) || {}
  $: urbitKeys = Object.keys(urbits)
</script>

{#if isOpen}
  <Modal width={800}>
    {JSON.stringify(urbitKeys)}
    <div class="wrapper">
      <h1>Report Bug</h1>
      <p>Submit a bug with your logs and we will contact you within 48 hours.</p>
      <h2>Contact Info</h2>
      <input placeholder="Email or Urbit Ship Name" bind:value={contact} />
      <h2>Describe Issue</h2>
      <textarea placeholder="Type here" bind:value={description} />
      <h2>Additional Information (Optional)</h2>
      <button
        on:click={()=>submitReport(contact,description,[])}
        >Send
      </button>
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
  }
  h2 {
    color: var(--Gray-400, #5C7060);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 20px;
    font-style: normal;
    font-weight: 300;
    letter-spacing: -1.2px;
  }
  input {
    flex: 1;
    color: var(--NP_Black, #313933);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    letter-spacing: -1.44px;
    border-radius: 16px;
    background: var(--Gray-100, #DDE3DF);
    padding: 16px 24px 18px 24px;
    border: none;
    width: calc(100% - 48px);
  }
  input:focus {
    outline: none;
  }
  input:focus {
    outline: none;
  }
  input:disabled {
    opacity: .6;
    pointer-events: none;
  }
  textarea {
    flex: 1;
    color: var(--NP_Black, #313933);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    letter-spacing: -1.44px;
    border-radius: 16px;
    background: var(--Gray-100, #DDE3DF);
    padding: 16px 24px 18px 24px;
    border: none;
    width: calc(100% - 48px);
    height: 120px;
    resize: none;
  }
  textarea:focus {
    outline: none;  
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
  button:disabled {
    pointer-events: none;
    opacity: .6;
  }
</style>
