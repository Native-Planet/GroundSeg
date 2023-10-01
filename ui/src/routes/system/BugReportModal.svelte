<script>
  import { afterUpdate } from 'svelte'
  import { structure, submitReport } from '$lib/stores/websocket'
  import { closeModal } from 'svelte-modals'
  import Modal from '$lib/Modal.svelte'
  export let isOpen
  let contact = ''
  let description = ''
  let cpuProfile = false
  let all = false
  const selectedShips = new Set()
  $: urbits = ($structure?.urbits) || {}
  $: urbitKeys = Object.keys(urbits)

  $: tBugReport = ($structure?.system?.transition?.bugReport) || ""
  $: tBugReportError = ($structure?.system?.transition?.bugReportError) || ""
  $: sendCondition = (contact.length == 0) || (description.length == 0) || (tBugReport == "loading")

  const toggleShipLog = patp => {
    if (selectedShips.has(patp)) {
      selectedShips.delete(patp)
    } else {
      selectedShips.add(patp) 
    }
  }

  afterUpdate(()=>{
    if (tBugReport == "done") {
      closeModal()
    }
  })

  const toggleSelectAll = () => {
    if (selectedShips.size == urbitKeys.length) {
      selectedShips.clear()
      all = false
    } else {
      urbitKeys.forEach(patp => selectedShips.add(patp));
      all = true
    }
  }
</script>

{#if isOpen}
  <Modal width={800}>
    <div class="wrapper">
      <h1>Report Bug</h1>
      <p>Submit a bug with your logs and we will contact you within 48 hours.</p>
      <h2>Contact Info</h2>
      <input placeholder="Email or Urbit Ship Name" bind:value={contact} />
      <h2>Describe Issue</h2>
      <textarea placeholder="Type here" bind:value={description} />
      <h2>Pier Logs(Optional)</h2>
      {#if urbitKeys.length > 0}
        <div class="ship-logs">
          {#each urbitKeys as p}
            <div class="check-wrapper" on:click={()=>toggleShipLog(p)}>
              <div class="checkbox">
                {#if Array.from(selectedShips).includes(p)}
                  <img class="checkmark" src="/checkmark.svg" alt="checkmark"  />
                {/if}
              </div>
              <div class="check-label">{p}</div>
            </div>
          {/each}
          {#if urbitKeys.length > 0}
            <div class="check-wrapper" on:click={toggleSelectAll}>
              <div class="checkbox">
                {#if all}
                  <img class="checkmark" src="/checkmark.svg" alt="checkmark"  />
                {/if}
              </div>
              <div class="check-label">{ all ? "Unselect" : "Select"} All</div>
            </div>
          {/if}
        </div>
      {/if}
      <h2>GroundSeg Deep Profile (Takes additional 30 Seconds)</h2>
      <div class="check-wrapper" on:click={()=>cpuProfile=!cpuProfile}>
        <div class="checkbox">
          {#if cpuProfile}
            <img class="checkmark" src="/checkmark.svg" alt="checkmark"  />
          {/if}
        </div>
        <div class="check-label">Get GroundSeg Deep Profile</div>
      </div>
      {#if tBugReportError.length > 0}
        <div class="error">{tBugReportError}</div>
      {:else}
        <button
          disabled={sendCondition}
          on:click={()=>submitReport(contact,description,Array.from(selectedShips),cpuProfile)}
          >
          {#if tBugReport == "success"}
            Success!
          {:else}
          Send{tBugReport == "loading" ? "ing..." : ""}
          {/if}
        </button>
      {/if}
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
  .ship-logs {
    display: flex;
    flex-wrap: wrap;
    gap: 16px;
  }
  .check-wrapper {
    display: flex;
    gap: 16px;
    cursor: pointer;
    width: calc(33% - 16px);
    user-select: none; /* Standard syntax */
    -webkit-user-select: none; /* Safari */
    -moz-user-select: none; /* Firefox */
    -ms-user-select: none; /* IE/Edge */
  }
  .checkbox {
    width: 16px;
    height: 16px;
    border-radius: 4px;
    border: 2px solid var(--Gray-200, #ABBAAE);
  }
  .checkmark {
    width: 14px;
    height: 14px;
    padding: 1px;
  }
  .check-label {
    color: #000;
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 16px;
    font-style: normal;
    font-weight: 300;
    letter-spacing: -1.44px;
  }
  .error {
    margin-top: 56px;
    border-radius: 16px;
    cursor: pointer;
    padding: 0 48px;
    height: 65px;
    color: red;
    text-align: center;
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    line-height: 32px; /* 133.333% */
    letter-spacing: -1.44px;
    text-align: left;
  }
</style>
