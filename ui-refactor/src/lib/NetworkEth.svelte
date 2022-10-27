<script>
  import { api } from '$lib/api'

  export let ethOnly

  const toggleEth = () =>  {
    let module = 'ethernet'
	  fetch($api + '/system?module=' + module, {
			method: 'POST',
			headers: {'Content-Type': 'application/json'},
			body: JSON.stringify({'action':'toggle'})
	  })
      .then(d => d.json())
      .then(res => console.log(res))
  }

</script>

<div class="ethernet">
  <div class="ethernet-text" class:disabled={ethOnly}>Ethernet Only</div>
  <div on:click={toggleEth} class="switch-wrapper">
    <div class="switch {ethOnly ? "on" : "off"}"></div>
  </div>
</div>

<style>
  .ethernet-text {
    font-size: 14px;
    flex: 1;
  }
  .switch-wrapper {
    border-radius: 8px;
    width: 32px;
    height: 12px;
    background: #ffffff4d;
    padding: 2px;
    cursor: pointer;
  }
  .switch {
    height: 100%;
    width: 19px;
    border-radius: 6px;
    margin-top: auto;
  }
  .on {
    background: #008eff;
    float: right;
  }
  .off {
    background: #000;
    float: left;
    opacity: .2;
  }
</style>
