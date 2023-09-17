<script>
  import { onMount, afterUpdate } from 'svelte'
  import { structure, startramGetRegions, startramRegister } from '$lib/stores/websocket'
  import { closeModal } from 'svelte-modals'
  import Modal from '$lib/Modal.svelte'
  export let regionMode = false
  export let isOpen
  let key = ''
  let selected = 'us-east'
  $: info = ($structure?.profile?.startram?.info) || {}
  $: transition = ($structure?.profile?.startram?.transition) || {}
  $: regions = info?.regions || {}
  $: regionKeys = Object.keys(regions)
  $: tRegister = (transition?.register) || null
  $: urbits = ($structure?.urbits) || {}
  $: urbitKeys = Object.keys(urbits)
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
  <Modal width={800}>
    <!--
    {#each urbitKeys as p}
      <div>{p} {JSON.stringify(urbits[p].transition.serviceRegistrationStatus)}</div>
    {/each}
    -->
    {#if tRegister == null}
      <div class="wrapper">
        <h1>{regionMode ? "Change Region" : "Register New Key"}</h1>
        <p>Entering a new key will replace the current one</p>
        <h2>{regionMode ? "Your" : "New"} Key</h2>
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
    line-height: 48px; /* 200% */
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
  .regions {
    display: flex;
    gap: 12px;
    justify-content: center;
    margin-top: 8px;
  }
  .region {
    flex: 1;
    text-align: center;
    background: var(--bg-modal);
    color: var(--btn-secondary);
    border-radius: 16px;
    font-size: 18px;
    height: 65px;
    line-height: 65px;
    cursor: pointer;
  }
  .highlight {
    color: var(--text-card-color);
    background: var(--btn-secondary);
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
