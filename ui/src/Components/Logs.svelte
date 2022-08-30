<script>
  import Fa from 'svelte-fa'
  import { faChevronDown } from '@fortawesome/free-solid-svg-icons/index.es'
</script>

  <div class="network">
    <div class="network-title">Export Logs</div>
    <select>
      <option>dister-dozzod-donpub-datdux</option>
    </select>
    <div class="buttons">
      <a class="view" href="#">View</a>
      <button class="export">Export</button>
    </div>
  </div>

<style>
  .network {
    display: flex;
    flex-direction: column;
    background: #0000006d;
    width: 300px;
    padding: 40px;
    border-radius: 15px;
    font-size: 18px;
    gap: 12px;
  }
  .network-title {
    font-size: 18px;
    padding-bottom: 8px;
  }
  select {
    border: none;
    background: #ffffff4d;
    color: inherit;
    padding: 12px;
    font-size: 14px;
    font-weight: 700;
    border-radius: 8px;
    appearance: none;
    background-image: url("/chevron-down-solid.svg");
    background-repeat: no-repeat;
    background-position: top 50% right 12px;
    background-size: 13px;
    margin-bottom: 8px;
  }
  select:focus {
    outline: none;
  }
  .buttons {
    display: flex;
  }
  .view {
    background: #ffffff4d;
    padding: 8px;
    font-size: 14px;
    border-radius: 6px;
    width: 60px;
    text-align: center;
  }
  .export {
    font-weight: 700;
    margin-left: auto;
    background: #008eff;
    padding: 8px;
    font-size: 14px;
    border-radius: 6px;
    width: 80px;
    text-align: center;
    cursor: pointer;
    border: none;
    color: white;
  }
</style>
