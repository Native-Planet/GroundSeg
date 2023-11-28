<script>
  import { afterUpdate } from 'svelte'
  import ToggleButton from '$lib/ToggleButton.svelte'
  import Fa from 'svelte-fa'
  import { faMinus, faPlus, faAngleUp, faAngleDown } from '@fortawesome/free-solid-svg-icons';
  import  { structure, installPenpaiCompanion, uninstallPenpaiCompanion, setPenpaiModel, setPenpaiCores, togglePenpai, removePenpai } from '$lib/stores/websocket'

  // TODO: onMount check desks, bypassing flood control

  let showModels = false

  // debug
  let installed = false

  $: urbits = ($structure?.urbits) || {}
  $: urbitKeys = Object.keys(urbits)
  $: penpai = ($structure?.apps?.penpai?.info) || {}
  $: models = (penpai?.models) || []
  $: activeModel = (penpai?.activeModel) || ""
  $: penpaiAllowed = (penpai?.allowed) || false
  $: penpaiRunning = (penpai?.running) || false
  $: minCores = 1
  $: maxCores = (penpai?.maxCores) || 1
  $: activeCores = (penpai?.activeCores) || 1

  let selectedModel = ""

  afterUpdate(()=>{
    if (selectedModel.length < 1) {
      selectedModel = activeModel
    }
  })

  const selectModel = model => {
    showModels = false
    selectedModel = model
  }

  const handleChangeModel = () => {
    setPenpaiModel(selectedModel)
  }

  const handlePenpaiCompanion = p => {
    if (urbits?.[p]?.info?.penpaiCompanion) {
      uninstallPenpaiCompanion(p)
    } else {
      installPenpaiCompanion(p)
    }
  }

</script>

<div class="container">
      <div class="sys-toggle">
        <div class="sys-title">PENPAI {!penpaiAllowed ? "(DISABLED)" : ""}</div>
        {#if penpaiAllowed}
            <ToggleButton
              on:click={togglePenpai}
              on={penpaiRunning}
              />
        {/if}
      </div>

  {#if penpaiAllowed}
    <div class="wifi-toggle">
      <div class="install-text">Allocate CPU Cores</div>
      <div class="right">
        <button disabled={activeCores == minCores} class="btn" on:click={()=>setPenpaiCores(activeCores - 1)}>
          <Fa icon={faMinus} size="1x" />
        </button>
        <div class="val">{activeCores} / {maxCores} Core{activeCores > 1 ? "s" : ""}</div>
        <button disabled={activeCores == maxCores} class="btn" on:click={()=>setPenpaiCores(activeCores + 1)}>
          <Fa icon={faPlus} size="1x" />
        </button>
      </div>
    </div>

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

    {#if selectedModel != activeModel}
      <div class="submit-buttons">
          <button class="connect" on:click={handleChangeModel}>Change Model</button>
      </div>
    {/if}

    {#if urbitKeys.length > 0}
      <div class="companion-title">Install Companion App</div>
      <div class="companion-wrapper">
        {#each urbitKeys as p}
          <div class="wifi-toggle" class:disable={!urbits?.[p]?.info?.running} on:click={()=>handlePenpaiCompanion(p)}>
            {#if urbits?.[p]?.info?.penpaiInstalling}
              <div class="loading-box"></div>
            {:else}
              <div class="checkbox">
                {#if urbits?.[p]?.info?.penpaiCompanion}
                  <img class="checkmark" src="/checkmark.svg" alt="checkmark"/>
                {/if}
              </div>
            {/if}
            <div class="companion-text">{p}</div>
          </div>
        {/each}
      </div>
    {/if}
    <!--
    <button class="remove" on:click={removePenpai}>Delete Penpai Local Data</button>
    -->
  {/if}
</div>

<style>
  .container {
    margin: 0;
  }
  .sys-title {
    flex: 1;
  }
  .sys-toggle {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 24px;
    margin-bottom: 55px;
  }
  .wifi-toggle {
    display: flex;
    align-items: center;
    gap: 24px;
  }
  input[type="range"] {
    flex: 1;
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
    margin-top: 32px;
  }
  .companion-wrapper {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  .install-text {
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
  }
  .checkbox {
    width: 24px;
    height: 24px;
    border: solid 2px var(--text-color);
    border-radius: 8px;
    cursor: pointer;
  }
  .loading-box {
    width: 18px;
    height: 18px;
    border-top: solid 5px var(--text-color);
    border-bottom: solid 5px var(--text-color);
    border-left: solid 5px #00000000;
    border-right: solid 5px #00000000;
    border-radius: 50%;
    cursor: pointer;
    animation: rotate 1s linear infinite;
  }
  .companion-text {
    color: var(--NP_Black, #161D17);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 21px;
    font-style: normal;
    font-weight: 300;
    line-height: 48px; /* 200% */
    letter-spacing: -1.44px;
    cursor: pointer;
    user-select: none;
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
  .remove {
    margin-top: 32px;
    background-color: black;
  }
  .connect {
    background-color: var(--btn-primary);
  }
  .disabled {
    opacity:.6;
    pointer-events: none;
  }
  .right {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 24px;
  }
  .btn {
    background: var(--btn-secondary);
    color: var(--text-card-color);
    text-align: center;
    height: 56px;
    line-height: 38px;
    border-radius: 16px;
    font-size: 20px;
    cursor: pointer;
    padding-bottom: 4px;
  }
  .btn:disabled {
    opacity: 0.6;
    pointer-events: none;
  }
  .val {
    color: #000;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    line-height: 24px; /* 100% */
    letter-spacing: -1.44px;
  }
  .checkmark {
    width: 16px;
    height: 16px;
    padding: 4px;
  }
  .disable {
    opacity: .6;
    pointer-events: none;
  }
  @keyframes rotate {
    0% { transform: rotate(0deg); }
    100% { transform: rotate(360deg); }
  }
</style>
