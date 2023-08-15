<script>
  import { wide } from '$lib/stores/display';
  import { bootShip, structure } from '$lib/stores/websocket';
  import { goto } from '$app/navigation';
  import KeyDropper from './KeyDropper.svelte';

  let key = '';
  let name = '';
  let remote = true;

  $: shipReady = ($structure?.newShip) || null
</script>

{JSON.stringify($structure?.newShip)}
<div id="card-wrapper" class="card-wrapper {wide ? "wide" : "slim"}">
  <div class="title">NEW SHIP</div>
  <div class="sigil"></div>
  <div class="input-wrapper">
    <div class="label">Urbit ID</div>
    <input type="text" bind:value={name} placeholder="Ship Name" />
  </div>
  <div class="input-wrapper">
    <div class="label">Bootfile</div>
    <KeyDropper on:change={e=> key = e.detail} />
  </div>
  <div class="check-wrapper" on:click={()=>remote = !remote}>
    <div class="checkbox" class:highlight={remote} ></div>
    <div class="check-label">Set to remote</div>
  </div>
  <div class="buttons">
    <button class="back" on:click={()=>goto('/boot')}>Back</button>
    <button class="boot" on:click={()=>bootShip(name,key,remote)} disabled={(key.length < 1) && (name.length < 1)}>Boot</button>
    <div class="spacer"></div>
  </div>
</div>

<style>
  .wide {
    width: 1104px;
    max-width: 100vw;
  }
  .slim {
    width: calc(100vw - 40px);
  }
  .card-wrapper {
    background: var(--bg-base);
    border-radius: 16px;
    margin: auto;
    height: 70vh;
    display:flex;
    flex-direction: column;
    align-items: center;
  }
  .slim .card-wrapper {
    background: var(--bg-base);
    border-radius: 16px;
    margin: auto;
    height: 70vh;
    display:flex;
    gap: 40px;
    flex-direction: column;
    align-items: center;
  }
  .title {
    font-family: var(--title-font);
    font-size: 48px;
    padding-top: 40px;
    padding-bottom: 40px;
  }
  .sigil {
    height: 160px;
    width: 160px;
    border: solid 4px var(--text-color);
    border-radius: 24px;
    background: var(--bg-modal);
    margin-bottom: 20px;
  }
  .input-wrapper {
    width: 60%;
    display: flex;
    flex-direction: column;
  }
  .check-wrapper {
    cursor: pointer;
    user-select: none;
    width: 60%;
    display: flex;
    gap: 12px;
    align-items: start;
  }
  .checkbox {
    width: 20px;
    height: 20px;
    border: solid 1px var(--btn-secondary);
    border-radius: 6px;
  }
  .highlight {
    background-color: var(--btn-secondary);
  }
  .check-label {
    line-height: 20px;
    font-size: 12px;
    margin-bottom: 24px;
  }
  .label {
    font-size: 12px;
    margin-bottom: 8px;
  }
  input {
    font-family: var(--regular-font);
    color: var(--text-color);
    padding-left: 20px;
    border: 2px solid var(--btn-secondary);
    background-color: var(--bg-modal);
    border-radius: 12px;
    font-size: 16px;
    line-height: 36px;
    margin-bottom: 20px;
  }
  input:focus {
    outline: none;
  }
  .buttons {
    width: 60%;
    display: flex;
    gap: 20px;
  }
  .spacer {
    flex: 2;
  }
  .back {
    font-family: var(--regular-font);
    color: var(--text-card-color);
    cursor: pointer;
    line-height: 38px;
    flex: 1;
    border-radius: 12px;
    background-color: var(--btn-secondary);
  }
  .boot {
    font-family: var(--regular-font);
    color: var(--text-card-color);
    cursor: pointer;
    line-height: 38px;
    flex: 1;
    border-radius: 12px;
    background-color: var(--btn-primary);
  }
  .back:hover {
    background-color: var(--bg-card);
  }

  .boot:hover {
    background-color: var(--bg-card);
  }
  .boot:disabled {
    opacity: .6;
    pointer-events: none;
  }

</style>
