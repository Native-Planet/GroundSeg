<script>
  // Modals
  import { openModal } from 'svelte-modals'
  import RegisterModal from './RegisterModal.svelte'
  // Icons
  import Fa from 'svelte-fa'
  import { faArrowRight } from '@fortawesome/free-solid-svg-icons';
  // Websocket
  import { startramRestart, startramToggle } from '$lib/stores/websocket'
  import { structure } from '$lib/stores/data'
  // Components
  import ToggleButton from '$lib/ToggleButton.svelte'
  import StarTramInfo from './StarTramInfo.svelte'
  import StarTramModify from './StarTramModify.svelte'

  // Info
  $: info = ($structure?.profile?.startram?.info) || {}
  $: renew = (info?.renew) || false
  $: expiry = (info?.expiry) || ""
  $: running = (info?.running) || false
  $: registered = (info?.registered) || false

  // Transition
  $: transition = ($structure?.profile?.startram?.transition) || {}
  $: tToggle = (transition?.toggle) || null
  $: tRestart = (transition?.restart) || ""
</script>

<div class="container">
  <div class="top">
    <StarTramInfo {renew} {expiry} {registered} />
    <div class="spacer"></div>
    <div class="controls">
      <div class="top">
        <button
          on:click={()=>openModal(RegisterModal)}
          class="new-key">
          New Key
        </button>
        {#if registered}
            <ToggleButton on:click={startramToggle} loading={tToggle} on={running} />
        {/if}
      </div>
      {#if running}
        <button
          disabled={tRestart.length > 0}
          on:click={startramRestart}
          class="restart"
          class:restarting={(tRestart.length > 0) && (tRestart != "done")}
          class:success={tRestart == "done"}
          >
          {#if tRestart == "startram"}
            Restarting StarTram
          {:else if tRestart == "urbits"}
            Fixing Urbit Ships
          {:else if tRestart == "minios"}
            Fixing MinIO containers
          {:else if tRestart == "done"}
            Success!
          {:else}
            Restart
          {/if}
        </button>
      {/if}
    </div>
  </div>
  <StarTramModify />
</div>

<style>
  .container {
    margin: auto;
    width: calc(1104px - (56px * 2));
    max-width: 98vw;
    padding: 56px;
  }
  .top {
    display: flex;
    gap: 24px;
  }
  .new-key {
    background: var(--btn-secondary);
    height: 65px;
    padding: 0 48px;
    border-radius: 16px;
    font-size: 24px;
    color: #FFF;
    text-align: center;
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    letter-spacing: -1.44px;
  }
  .restart {
    background: black;
    height: 65px;
    padding: 0 48px;
    border-radius: 16px;
    color: #FFF;
    text-align: center;
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    letter-spacing: -1.44px;
  }
  .restarting {
    animation: breathe 5s infinite;
    pointer-events: none;
  }
  .success {
    pointer-events: none;
    opacity: .6;
  }
  .controls {
    display: flex;
    flex-direction: column;
    align-items: end;
    gap: 20px;
  }
  .spacer {
    flex: 1;
  }
  button {
    cursor: pointer;
  }
  @keyframes breathe {
    0% {
      opacity: .2;
    }
    50% {
      opacity: .6;
    }
    100% {
      opacity: .2;
    }
  }
</style>
