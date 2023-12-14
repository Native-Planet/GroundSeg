<script>
  // Style
  import "../theme.css"
  import { openModal } from 'svelte-modals'
  import LoginModal from '../loginmodal.svelte'
  import { structure } from '$lib/stores/data'
  import { URBIT_MODE } from '$lib/stores/data'
  $: pfx = $URBIT_MODE ? "/apps/groundseg" : ""


  const handleModal = () => {
    openModal(LoginModal,{})
  }

</script>

<div class="section">
  <div class="section-left">
    <div class="section-title">Admin Mode</div>
    <div class="section-description">
      Login using admin password to access GroundSeg with admin privileges
    </div>
  </div>
  <div class="section-right">
    <div class="btn-wrapper">
      <div class="spacer"></div>
      <button class="start" on:click={handleModal}>
          Login
      </button>
    </div>
  </div>
</div>

<style>
  .btn-wrapper {
    display: flex; 
    gap: 8px;
  }
  .spacer {
    flex: 1;
  }
  button {
    cursor: pointer;
  }
  button:disabled {
    opacity: .6;
    pointer-events: none;
  }
  .start {
    display: flex;
    padding: 20px 48px;
    justify-content: center;
    align-items: center;
    gap: 8px;
    border-radius: 16px;
    background: #2C3A2E;
    color: #FFF;
    font-family: Inter;
    font-size: 32px;
    font-style: normal;
    font-weight: 300;
    line-height: 24px; /* 75% */
    letter-spacing: -1.92px;
  }
</style>
