<script>
  import KeyDropper from './KeyDropper.svelte';
  import { bootShip } from '$lib/stores/websocket';
  import { structure } from '$lib/stores/data'
  import { sigRemove, checkPatp } from '$lib/stores/patp';
  import { goto } from '$app/navigation';
  import Sigil from './Sigil.svelte'
  import { URBIT_MODE } from '$lib/stores/data'
  import Fa from 'svelte-fa'
  import { faCircleExclamation, faAngleUp, faAngleDown } from '@fortawesome/free-solid-svg-icons';

  import { openModal } from 'svelte-modals'
  import NewDriveWarning from './NewDriveWarning.svelte'

  $: pfx = $URBIT_MODE ? "/apps/groundseg" : ""

  let key = '';
  let name = '';
  let remote = true;
  let advanceOpen = false
  let selectedDrive = "system-drive"

  $: noSig = sigRemove(name)
  $: validPatp = checkPatp(noSig)

  $: registered = ($structure?.profile?.startram?.info?.registered) || false
  $: running = ($structure?.profile?.startram?.info?.running) || false

  $: drives = $structure?.system?.info?.drives || {}
  $: driveNames = Object.keys(drives)

  const handleBoot = () => {
      bootShip(noSig,key,remote,selectedDrive) // temp
    /*
    if (selectedDrive == "system-drive") {
      bootShip(noSig,key,remote,selectedDrive)
    } else {
      openModal(NewDriveWarning)
    }
    */
  }
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
  <KeyDropper
    on:changeKey={e => key = e.detail}
    on:changePatp={e => name = e.detail}
  />
</div>


<!-- Customize -->
<div class="input-wrapper">
  <div class="advance" on:click={()=>advanceOpen = !advanceOpen}>
    Customize <Fa icon={advanceOpen ? faAngleUp : faAngleDown} size="1x" />
  </div>
</div>
{#if advanceOpen}
<div class="input-wrapper">
  <div class="label">Select Drive</div>
  <div class="mount-wrapper">
    <div class="mount-info" on:click={()=>selectedDrive="system-drive"} class:active={selectedDrive=="system-drive"}>System Drive (default)</div>
    {#each driveNames as name}
      <div class="mount">
      <div
        class="mount-info"
        class:active={selectedDrive==name}
        on:click={()=>selectedDrive=name}
        >{drives[name].driveID == 0 ? "New Drive" : "Drive " + drives[name].driveID} ({name})
      </div>
      <div class="mount-icon" on:click={()=>openModal(NewDriveWarning,{driveName:name})}>
        <Fa icon={faCircleExclamation} size="1.5x" />
      </div>
    </div>
    {/each}
  </div>
</div>
<div class="input-wrapper">
  <div class="label">Configuration</div>
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
</div>
{/if}
<div class="input-wrapper">
  <div class="buttons">
    <button class="btn back" on:click={()=>goto(pfx+'/boot')}>Back</button>
    <button
      class="btn boot"
      on:click={handleBoot}
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
    margin-top: 16px;
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
  .advance {
    cursor: pointer;
    color: var(--Gray-400, #5C7060);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 32px;
    font-style: normal;
    font-weight: 300;
    letter-spacing: -1.44px;
    padding-top: 16px;
  }
  .mount-wrapper {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }
  .mount {
    display: flex;
    gap: 16px;
    align-items: center;
  }
  .mount-info {
    padding: 16px;
    border: solid 2px var(--btn-secondary);
    border-radius: 16px;
    width: 200px;
    text-align: center;
    cursor: pointer;
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 18px;
    font-style: normal;
    letter-spacing: -1.44px;
    user-select: none;
  }
  .mount-icon {
    color: orange;
    cursor: pointer;
  }
  .active {
    background: var(--btn-secondary);
    color: white;
    pointer-events: none;
  }
</style>
