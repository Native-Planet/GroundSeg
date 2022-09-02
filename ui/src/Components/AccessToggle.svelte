<script>
  import { url } from '/src/Scripts/server'
  import Fa from 'svelte-fa'
  import { faArrowUpRightFromSquare } from '@fortawesome/free-solid-svg-icons/index.es'

  export let name, nw_label
  let isSwitching = false

  const toggleNetwork = () => { 

    isSwitching = true
    let u = url + "/urbit/network"
    const f = new FormData()
    f.append(name,'network')

    fetch(u, {method: 'POST',body: f})
      .then(r => r.json())
      .then(d => { if (d == 200) {
        isSwitching = false
   }})}

</script>

    <div class="info" class:switching={isSwitching} on:click={toggleNetwork}>
      <div class="title">Access</div>
      <div class="access-options">
        <button class="option" class:access-active={nw_label === 'Local'} >Local</button>
        <button class="option" class:access-active={nw_label === 'Remote'} >Remote</button>
      </div>
    </div>

<style>
  .info {
    margin-bottom: 12px;
  }
  .title {
    font-weight: 700;
    margin-bottom: 12px;
    text-align: left;
  }

  .access-options {
    display: flex;
    width: 240px;
    border-radius: 8px;
    background: #ffffff4d;
    gap: 2px;
  }
  .option {
    color: inherit;
    font-size: 14px;
    flex: 1;
    padding: 8px 0 8px 0;
    background: none;
    border-radius: 8px;
    border: none;
    font-weight: 700;
    cursor: pointer;
  }
  .switching {
    opacity: .6;
    pointer-events: none;
  }
  .access-active {
    background: #008eff;
  }

</style>
