<script>
  import { onMount, afterUpdate } from 'svelte'
  import { structure, startramGetRegions, startramRegister } from '$lib/stores/websocket'
  import { closeModal } from 'svelte-modals'
  import Modal from '$lib/Modal.svelte'
  export let isOpen
  let key = ''
  let selected = 'us-east'
  $: info = ($structure?.profile?.startram?.info) || {}
  $: transition = ($structure?.profile?.startram?.transition) || {}
  $: regions = info?.regions || {}
  $: regionKeys = Object.keys(regions)
  $: tRegister = (transition?.register) || null
  let completed = false

  onMount(()=>startramGetRegions())
  afterUpdate(()=>{
    if (completed) {
      if (tRegister == null) {
        closeModal()
      }
    } else {
      if (tRegister == "complete") {
        completed = true
      }
    }
  })
</script>

{#if isOpen}
  <Modal>
    {#if tRegister == null}
      <div class="wrapper">
        <h1>Register New Key</h1>
        <p>Entering a new key will replace the current one</p>
        <h2>New Key</h2>
        <input disabled={tRegister != null} type="password" placeholder="NativePlanet-something-something" bind:value={key} />
        {#if regionKeys.length > 0}
          <h2>Select Region</h2>
          <div class="regions">
            {#each regionKeys as r}
              <div
                class="region"
                class:highlight={r == selected}
                on:click={()=>selected=r}>
                {regions[r].desc}
              </div>
            {/each}
          </div>
        {/if}
        <button
          disabled={(key.length < 1) || (tRegister != null)}
          on:click={()=>startramRegister(key,selected)}
          >Save
        </button>
      </div>
    {:else if tRegister == "key"}
      <div class="notify">Registering your key</div>
    {:else if tRegister == "services"}
      <div class="notify">Registering services</div>
    {:else if tRegister == "starting"}
      <div class="notify">Configuring your StarTram client</div>
    {:else if tRegister == "complete"}
      <div class="success">StarTram Key Registration Successful!</div>
    {:else}
      {tRegister}
    {/if}
  </Modal>
{/if}

<style>
  .wrapper {
    margin: 20px;
    font-family: var(--regular-font);
  }
  h1 {
    font-size: 14px;
    font-weight: 500;
  }
  p {
    margin-top: 10px;
    font-size: 14px;
    font-weight: 300;
    margin-bottom: 20px;
    opacity: .8;
  }
  h2 {
    padding-left: 8px;
    font-size: 14px;
    font-weight: 300;
    margin: 0;
    color: var(--btn-secondary);
    font-weight: 500;
  }
  input {
    margin: 8px 0 20px 0;
    background: var(--bg-modal);
    border-radius: 12px;
    width: calc(100% - 24px);
    border: none;
    padding: 12px;
  }
  input:focus {
    outline: none;
  }
  input:disabled {
    opacity: .6;
    pointer-events: none;
  }
  .regions {
    display: flex;
    gap: 12px;
    justify-content: center;
    margin-top: 8px;
  }
  .region {
    flex: 1;
    text-align: center;
    padding: 12px 0;
    background: var(--bg-modal);
    color: var(--btn-secondary);
    border-radius: 12px;
    font-size: 12px;
    font-weight: 600;
    cursor: pointer;
  }
  .highlight {
    color: var(--text-card-color);
    background: var(--btn-secondary);
  }
  button {
    margin-top: 48px;
    background-color: var(--btn-secondary);
    border-radius: 12px;
    color: var(--text-card-color);
    height: 42px;
    padding: 0 64px;
    font-family: var(--regular-font);
    font-size: 12px;
    cursor: pointer;
  }
  button:disabled {
    pointer-events: none;
    opacity: .6;
  }
  .notify {
    padding: 120px 0px;
    font-size: 24px;
    text-align: center;
    animation: breathe 5s infinite;
    color: var(--btn-secondary);
  }
  .success {
    padding: 120px 0px;
    font-size: 24px;
    text-align: center;
    color: var(--btn-primary);
  }
  @keyframes breathe {
    0% {
      opacity: .2;
    }
    50% {
      opacity: 1;
    }
    100% {
      opacity: .2;
    }
  }
</style>
