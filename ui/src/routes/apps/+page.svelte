<script>
  import './apps.css'
  import { structure } from '$lib/stores/data'
  import Penpai from './Penpai.svelte'
</script>

<div class="panel">
  <Penpai />
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
