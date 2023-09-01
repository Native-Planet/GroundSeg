<script>
  import KeyDropper from './KeyDropper.svelte';
  import { bootShip } from '$lib/stores/websocket';
  import { sigRemove, checkPatp } from '$lib/stores/patp';
  import { goto } from '$app/navigation';
  import Sigil from './Sigil.svelte'

  let key = '';
  let name = '';
  let remote = true;

  $: noSig = sigRemove(name)
  $: validPatp = checkPatp(noSig)

</script>

<div class="sigil-wrapper">
  <Sigil {name} reverse={true} />
</div>
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
  <button
    class="boot"
    on:click={()=>bootShip(noSig,key,remote)}
    disabled={
    (key.length < 1) || (name.length < 1) || (!validPatp)
    }>
    Boot
  </button>
  <div class="spacer"></div>
</div>

<style>
  .sigil-wrapper {
    width: 160px;
    height: 160px;
    border: solid 4px var(--text-color);
    border-radius: 24px;
    overflow: hidden;
    margin-bottom: 20px;
  }
  .spacer {
    flex: 2;
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
  .check-label {
    line-height: 20px;
    font-size: 12px;
    margin-bottom: 24px;
  }
  .label {
    font-size: 12px;
    margin-bottom: 8px;
  }
  .buttons {
    width: 60%;
    display: flex;
    gap: 20px;
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
  .highlight {
    background-color: var(--btn-secondary);
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
</style>
