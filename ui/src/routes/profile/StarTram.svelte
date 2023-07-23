<script>
  import { structure } from '$lib/stores/websocket'
  import Fa from 'svelte-fa'
  import { faArrowRight } from '@fortawesome/free-solid-svg-icons';

  import NewRegistration from './NewRegistration.svelte'
  import StarTramInfo from './StarTramInfo.svelte'

</script>

<div class="container">
  <div class="title">STARTRAM</div>
  <StarTramInfo />
</div>

<style>
  .container {
    margin: auto;
    width: calc(1104px - 80px);
    max-width: 98vw;
    padding: 40px;
  }
  .title {
    margin-bottom: 24px;
  }
  .header {
    font-size: 12px;
    margin-bottom: 12px;
  }
  input {
    width: calc(100% - 20px);
    font-family: var(--regular-font);
    color: var(--text-color);
    padding-left: 20px;
    border: 2px solid var(--btn-secondary);
    background-color: var(--bg-modal);
    border-radius: 12px;
    font-size: 12px;
    line-height: 36px;
  }
  input:focus {
    outline: none;
  }
  .spacer {
    flex: 1;
  }
  .region-wrapper {
    flex: 8;
  }
  .body {
    display: flex;
    gap: 40px;
  }
  .reg-wrapper {
    flex: 9;
  }
  .reg {
    display: flex;
    gap: 20px;
    margin-bottom: 20px;
  }
  .reg-btn {
    background-color: var(--btn-primary);
    color: var(--text-card-color);
    padding: 0 40px;
    border-radius: 16px;
  }
  .regions {
    display: flex;
    flex-wrap: wrap;
    gap: 24px;
  }
  .region {
    flex: 1 0 calc(50% - 20px);
    color: var(--text-card-color);
    border-radius: 8px;
    background: var(--btn-secondary);
    font-size: 12px;
    line-height: 64px;
    text-align: center;
  }
  .highlight {
    background: var(--btn-primary);
  }
</style>
