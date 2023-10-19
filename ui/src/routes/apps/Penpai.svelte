<script>
  import { onMount } from 'svelte'
  import ToggleButton from '$lib/ToggleButton.svelte'
  import Fa from 'svelte-fa'
  import { faAngleUp, faAngleDown } from '@fortawesome/free-solid-svg-icons';
  import  { structure, setPenpaiModel } from '$lib/stores/websocket'

  let showModels = false

  // debug
  let status = false
  let installed = false

  $: urbits = ($structure?.urbits) || {}
  $: urbitKeys = Object.keys(urbits)
  $: penpai = ($structure?.apps?.penpai?.info) || {}
  $: models = (penpai?.models) || []
  $: activeModel = (penpai?.activeModel) || ""
  $: penpaiAllowed = (penpai?.allowed) || false

  let selectedModel = ""
  onMount(()=>selectedModel = activeModel)

  const selectModel = model => {
    showModels = false
    selectedModel = model
  }

  const handleChangeModel = () => {
    setPenpaiModel(selectedModel)
  }

</script>

<div class="container">
  <div class="sys-title">PENPAI {!penpaiAllowed ? "(DISABLED)" : ""}</div>

  {#if penpaiAllowed}
    <div class="wifi-options">
        <div class="active">
          <div class="active-title">Model</div>
          <div class="active-selector" on:click={()=>showModels = !showModels}>
            <div class="active-text">{selectedModel.length < 1 ? "Select a model" : selectedModel}</div>
            <div class="active-arrow">
              {#if showModels}
                <Fa icon={faAngleUp} size="1x" />
              {:else}
                <Fa icon={faAngleDown} size="1x" />
              {/if}
            </div>
          </div>
        </div>

        {#if showModels}
          <div class="networks">
            {#each models as n}
              <div class="network" on:click={()=>{selectModel(n)}}>{n}</div>
            {/each}
          </div>
        {/if}
    </div>

    <div class="submit-buttons">
      {#if selectedModel != activeModel}
        <button class="connect" on:click={handleChangeModel}>Change Model</button>
      {/if}
    </div>

    <div class="companion-title">Urbit Companion App</div>
    <div class="companion-wrapper">
      {#each urbitKeys as p}
        <div class="wifi-toggle">
            <div class="companion-text">{p}</div>
            {#if installed}
              <ToggleButton
                on:click={()=>status=!status}
                on={status}
                />
            {:else}
              <button class="connect">Install</button>
            {/if}
        </div>
      {/each}
    </div>
  {/if}
</div>

<style>
  .container {
    margin: 0;
  }
  .sys-title {
    margin-bottom: 56px;
  }
  .wifi-toggle {
    display: flex;
    align-items: center;
  }
  .wifi-options {
    display: flex;
    flex-direction: column;
  }
  .companion-title {
    flex: 1;
    color: var(--NP_Black, #161D17);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    line-height: 48px; /* 200% */
    letter-spacing: -1.44px;
    margin-bottom: 32px;
  }
  .companion-wrapper {
    display: flex;
    flex-direction: column;
    gap: 24px;
  }
  .companion-text {
    flex: 1;
    color: var(--NP_Black, #161D17);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 18px;
    font-style: normal;
    font-weight: 300;
    line-height: 48px; /* 200% */
    letter-spacing: -1.44px;
  }
  .active-title {
    color: var(--NP_Black, #161D17);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    letter-spacing: -1.44px;
    margin-bottom: 16px;
    margin-top: 32px;
  }
  .active-selector {
    display: flex;
    background: var(--bg-modal);
    align-items: center;
    border-radius: 16px;
    height: 65px;
    cursor: pointer;
  }
  .active-text {
    flex: 1;
    font-size: 13px;
    font-weight: 600;
    user-select: none;

    color: var(--NP_Black, #161D17);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    letter-spacing: -1.44px;
    padding: 20px;
  }
  .active-arrow {
    width: 40px;
  }
  .networks {
    margin-top: 16px;
    display: flex;
    flex-direction: column;
    background: var(--btn-secondary);
    padding: 20px 0;
    color: var(--text-card-color);
    border-radius: 16px;
  }
  .network {
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    letter-spacing: -1.44px;
    padding: 10px 20px;
    user-select: none;
  }
  .network:hover {
    background: var(--bg-card);
    
  }
  .submit {
    margin-top: 32px;
    display: flex;
    flex-direction: column;
  }
  input {
    font-family: var(--regular-font);
    color: var(--text-color);
    padding-left: 20px;
    border: 2px solid var(--btn-secondary);
    background-color: var(--bg-modal);
    border-radius: 16px;
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    letter-spacing: -1.44px;
    padding: 0 20px 0 20px;
    height: 61px;
  }
  input:focus {
    outline: none;
  }
  input:disabled {
    opacity: .6;
    pointer-events: none;
  }
  .submit-buttons {
    margin-top: 20px;
    display: flex;
    gap: 24px;
    margin-bottom: 56px;
  }
  button {
    border-radius: 16px;
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    letter-spacing: -1.44px;
    padding: 0 48px;
    color: var(--text-card-color);
    font-family: var(--regular-font);
    cursor: pointer;
    height: 65px;
  }
  button:disabled {
    opacity: .6;
    pointer-events: none;
  }
  .cancel {
    background-color: var(--btn-secondary);
  }
  .connect {
    background-color: var(--btn-primary);
  }
  .disabled {
    opacity:.6;
    pointer-events: none;
  }
</style>
