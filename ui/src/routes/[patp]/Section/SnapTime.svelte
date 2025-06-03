<script>
  import { openModal } from 'svelte-modals'
  import FinalModal from './FinalModal.svelte';
  import UnplugWarning from './UnplugWarning.svelte';
  import SnapSlider from './SnapSlider.svelte'
  export let patp
  export let snapTime
  export let ownShip
</script>

<div class="section">
  <div class="section-left">
    <div class="section-title">Snapshot time</div>
    <div class="section-description">
      How frequently your ship saves snapshots to disk. We default to 60 so that your disk doesn't wear down as quickly, but it will replay more slowly.
    </div>
  </div>
  <div class="section-right">
    <UnplugWarning component={"snapTime"} {ownShip} >
      <div class="slider">
        <div class="spacer"></div>
        <SnapSlider {patp} {snapTime}/>
      </div>
    </UnplugWarning>
  </div>
</div>

<style>
  .slider {
    display: flex;
  }
  .spacer {
    flex: 1;
  }
</style>