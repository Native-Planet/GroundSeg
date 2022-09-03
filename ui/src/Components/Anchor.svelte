<script>
  let key = '', view = false, key2

  const toggleView = () => {
    view = !view
    key2 = key

  }
</script>

  <div class="network">
    <div class="network-title">Anchor Settings</div>
    <div class="switch">
      <button class="toggle active">On</button>
      <button class="toggle">Off</button>
    </div>
    <div class="reg-key-wrapper">
      <div class="reg-title">Key Registration</div>
      <div class="reg-key">
        <input type="text" bind:value={key} />
        <img on:click={()=> view = !view} src="/eye-{view ? "closed" : "open"}.svg" alt="eye" />
      </div>
      <button class="submit" class:disabled={key == ''}>Submit</button>
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
  .switch {
    display: flex;
    gap: 6px;
    padding: 2px;
    border-radius: 6px;
    background: #ffffff4d;
    width: 90px;
    height: 30px;
  }
  .toggle {
    background: none;
    border: none;
    border-radius: 4px;
    color: inherit;
    flex: 1;
    padding: 2px;
  }
  .active {
    background: #008eff;
  }
  .reg-key-wrapper {
    display: flex;
    flex-direction: column;
    gap: 6px;
    margin-top: 12px;
  }
  .reg-title {
    font-size: 14px;
    padding-bottom: 4px;
  }
  .reg-key {
    display: flex;
  }
  .reg-key > input {
    font-family: inherit;
    background: #ffffff4d;
    color: inherit;
    border-radius: 6px;
    font-size: 12px;
    padding: 8px;
    border: none;
    flex: 1;
  }
  input:focus {
    outline: none;
  }
  .reg-key > img {
    padding-left: 12px;
  }
  .submit {
    background: #008eff;
    padding: 8px;
    color: inherit;
    border: none;
    border-radius: 6px;
    font-family: inherit;
  }
  .disabled {
    opacity: .6;
    pointer-events: none;
  }
</style>
