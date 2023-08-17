<script>
  import { onMount, afterUpdate } from 'svelte'
  import { structure, startramGetRegions, startramRegister } from '$lib/stores/websocket'
  import { showRegisterModal } from './store'

  let key = ''
  let activeRegion = "us-east";
  let success = false
  let spinner = false

  $: info = ($structure?.profile?.startram?.info) || {}
  $: registered = (info?.registered) || false
  $: regions = (info?.regions) || {}
  $: regionKeys = Object.keys(regions)

  $: transition = ($structure?.profile?.startram?.transition) || {}
  $: tRegister = (transition?.register) || null

  const setRegion = r => {
    activeRegion = r
  }
  onMount(()=>startramGetRegions())
  afterUpdate(()=>{
    spinner = tRegister == "loading"
    success = tRegister == "success"
    if (tRegister == "done") {
      showRegisterModal.set(false)
    }
  })
</script>

<div class="wrapper">
  <div class="modal">
    <div class="header">Register a{registered ? "nother":""} key</div>

    <div class="name">Activation Key</div>
    <div class="activate">
      <input placeholder="NativePlanet-some-word-another-word" type="password" bind:value={key}/>
    </div>
    {#if regionKeys.length > 0}
      <div class="name">Select Region</div>
      <div class="regions">
        {#each regionKeys as r }
          <div on:click={()=>setRegion(r)} class="region" class:highlight={r == activeRegion}>
            {regions[r].desc}
          </div>
        {/each}
      </div>
    {/if}
    <div class="buttons">
      <button
        class="btn-cancel"
        on:click={()=>showRegisterModal.set(false)}
        >Back
      </button>
      <button
        class="btn-activate"
        disabled={key.length < 1} 
        on:click={()=>startramRegister(key,activeRegion)}
        >
        {#if spinner}
          Loading
        {:else if success}
          Success!
        {:else}
          Activate
        {/if}
      </button>
      <div class="spacer"></div>
    </div>

    <a class="get" href="https://www.nativeplanet.io/startram" target="_blank">
      GET A STARTRAM KEY
    </a>

  </div>
</div>

<style>
  .wrapper {
    position:absolute;
    left: 0;
    top: 0;
    backdrop-filter: blur(4px);
    -moz-backdrop-filter: blur(4px);
    -o-backdrop-filter: blur(2px);
    -webkit-backdrop-filter: blur(4px);
    width: 100vw;
    height: 100vh;
    background: #FFFFFF3D;
  }
  .modal {
    display: flex;
    flex-direction: column;
    position: absolute;
    top: calc(50vh - (392px/2));
    left: calc(50vw - (572px/2));
    background: var(--bg-modal);
    width: calc(572px - 80px);
    height: calc(392px - 80px);
    border-radius: 16px;
    padding: 40px;
    gap: 12px;
  }
  .header {
    font-family: var(--regular-font);
    font-size: 14px;
  }
  .regions {
    display: flex;
    gap: 20px;
  }
  .region {
    font-size: 12px;
    font-family: var(--regular-font);
    color: var(--text-color);
    border: solid 2px var(--btn-secondary);
    border-radius: 12px;
    padding: 8px 0;
    text-align: center;
    flex: 1;
    cursor: pointer;
  }
  .highlight {
    color: var(--text-card-color);
    background-color: var(--btn-secondary);
  }
  .name {
    font-family: var(--regular-font);
    font-size: 12px;
    margin-top: 12px;
  }
  .activate {
    display: flex;
    gap: 20px;
  }
  .btn-activate {
    background: var(--btn-primary);
    padding: 0 20px;
    color: var(--text-card-color);
    border-radius: 12px;
  }
  .btn-cancel {
    background: var(--btn-secondary);
    padding: 8px 20px;
    color: var(--text-card-color);
    border-radius: 12px;
  }
  .btn-activate:disabled {
    background: var(--btn-secondary);
    color: var(--text-color);
    opacity: .6;
  }
  .buttons {
    margin-top: 30px;
    display: flex;
    height: 36px;
    gap: 20px;
  }
  button:hover {
    cursor: pointer;
  }
  button {
    font-family: var(--regular-font);
    flex: 1;
  }
  input {
    width: calc(100% - 24px);
    line-height: 36px;
    border: solid 2px var(--btn-secondary);
    border-radius: 12px;
    background: none;
    padding-left: 20px;
  }
  input:active, :focus {
    outline: none; 
  }
  .get {
    font-family: var(--title-font);
    position: absolute;
    bottom: 20px;
    color: var(--text-color);
    font-size: 14px;
    text-decoration: underline;
  }
  .spacer {
    flex: 1;
  }
</style>
