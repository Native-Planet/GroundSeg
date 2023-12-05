<script>
  import KeyDropper from './KeyDropper.svelte';
  import { bootShip } from '$lib/stores/websocket';
  import { structure } from '$lib/stores/data'
  import { sigRemove, checkPatp } from '$lib/stores/patp';
  import { goto } from '$app/navigation';
  import Sigil from './Sigil.svelte'
  import { URBIT_MODE } from '$lib/stores/data'
  $: pfx = $URBIT_MODE ? "/apps/groundseg" : ""

  let key = '';
  let name = '';
  let remote = true;

  $: noSig = sigRemove(name)
  $: validPatp = checkPatp(noSig)

  $: registered = ($structure?.profile?.startram?.info?.registered) || false
  $: running = ($structure?.profile?.startram?.info?.running) || false

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
  <div class="check-wrapper" on:click={()=>remote = !remote}>
    {#if registered && running}
      <div class="checkbox">
        {#if remote}
          <img class="checkmark" src={pfx+"/checkmark.svg"} alt="checkmark"/>
        {/if}
      </div>
      <div class="check-label">Set to remote</div>
    {/if}
  </div>
  <div class="buttons">
    <button class="btn back" on:click={()=>goto(pfx+'/boot')}>Back</button>
    <button
      class="btn boot"
      on:click={()=>bootShip(noSig,key,remote)}
      disabled={
      (key.length < 1) || (name.length < 1) || (!validPatp)
      }>
      Boot
    </button>
  </div>
</div>

<style>
  .sigil-wrapper {
    width: 128px;
    height: 128px;
    border-radius: 16px;
    overflow: hidden;
    margin: auto;
    margin: 0 auto 32px auto;
  }
  .input-wrapper {
    margin: auto;
    display: flex;
    width: 621px;
    padding-bottom: 0px;
    flex-direction: column;
    align-items: flex-start;
    gap: 16px;
    margin-bottom: 16px;
  }
  .check-wrapper {
    display: flex;
    justify-content: center;
    align-items: center;
    gap: 16px;
    cursor: pointer;
    user-select: none; /* Standard syntax */
    -webkit-user-select: none; /* Safari */
    -moz-user-select: none; /* Firefox */
    -ms-user-select: none; /* IE/Edge */
  }
  .checkbox {
    width: 28px;
    height: 28px;
    border-radius: 4px;
    border: 2px solid var(--Gray-200, #ABBAAE);
  }
  .checkmark {
    margin: 4px;
  }
  .check-label {
    color: #000;
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    letter-spacing: -1.44px;
  }
  .label {
    color: var(--Gray-400, #5C7060);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    letter-spacing: -1.44px;
  }
  .buttons {
    display: flex;
    gap: 16px;
    text-align: center;
  }

  .btn {
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    line-height: 32px; /* 133.333% */
    letter-spacing: -1.44px;
    color: #FFF;
  }
  .back {
    font-family: var(--regular-font);
    color: var(--text-card-color);
    cursor: pointer;
    background-color: var(--btn-secondary);
    border-radius: 16px;
    padding: 0 48px;
  }
  .boot {
    font-family: var(--regular-font);
    color: var(--text-card-color);
    cursor: pointer;
    border-radius: 16px;
    background-color: var(--btn-primary);
    height: 65px;
    padding: 0 48px;
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
  input:focus {
    outline: none;
  }
  input {
    flex: 1;
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    letter-spacing: -1.44px;
    border-radius: 16px;
    padding: 10px 22px 12px 22px;
    width: calc(100% - 48px);
    border: 2px solid var(--Gray-400, #5C7060);
    background: var(--bg-base);
    color: var(--text-color);

  }
  input::placeholder {
    color: var(--Gray-200, #ABBAAE);
  }
</style>
