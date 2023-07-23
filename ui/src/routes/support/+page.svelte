<script>
  import { wide } from '$lib/stores/display'
  import './system.css'
  import Interactive from './Interactive.svelte'
  import BugReport from './BugReport.svelte'
  import Support from './Support.svelte'
</script>

<div class="panel">
  <Interactive />
  <BugReport />
  <Support />
</div>

<style>
  .panel {
    width: 1104px;
    max-width: 98vw;
    margin: auto;
    height: auto;
    border-radius: 16px;
    display :flex;
    flex-direction: column;
    gap: 20px;
  }
</style>
