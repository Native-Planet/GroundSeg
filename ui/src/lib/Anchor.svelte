<script>
  let key = '', view = false, key2, isAnchorOff = false

  const toggleView = () => {
    view = !view
    document.querySelector('#input').type = view ? 'text' : 'password'
  }

  const toggleAnchor = () => {
    isAnchorOff = !isAnchorOff
  }

</script>

  <div class="anchor">

    <div class="title-wrapper">
      <div class="title">Anchor</div>
      <div on:click={toggleAnchor} class="switch-wrapper">
        <div class="switch {isAnchorOff ? "off" : "on"}"></div>
      </div>
    </div>

    <div class="reg-key-wrapper">
      <div class="reg-title">Key Registration</div>
      <div class="reg-key">
        <input id='input' type="password" bind:value={key} />
        <img on:click={toggleView} src="/eye-{view ? "closed" : "open"}.svg" alt="eye" />
      </div>
      <button class="submit" class:disabled={key == ''}>Submit</button>
    </div>

  </div>

<style>
  .anchor {
    background: #0000006d;
    width: 300px;
    padding: 40px;
    border-radius: 15px;
    font-size: 18px;
  }
  .title-wrapper {
    display: flex;
    align-items: center;
  }
  .title {
    font-size: 18px;
    flex: 1;
  }
  .switch-wrapper {
    border-radius: 8px;
    width: 32px;
    height: 12px;
    background: #ffffff4d;
    padding: 2px;
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
