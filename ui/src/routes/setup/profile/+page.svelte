<script>
  import { setupPassword } from '$lib/stores/websocket'
  let pwd = ''
  let cfm = ''

</script>

<div class="title">PASSWORD</div>
<div class="wrapper">
  <div class="pw-wrapper">
    <div class="name">Create Password</div>
    <input type="password" placeholder="Password" bind:value={pwd} />
  </div>
  <div class="pw-wrapper">
    <div class="name">Confirm Password</div>
    <input type="password" placeholder="Password" bind:value={cfm} />
  </div>
  <button
    on:click={()=>setupPassword(cfm)}
    disabled={(pwd.length == 0) || (pwd !== cfm)}
    >Continue
  </button>
</div>

<style>
  .wrapper {
    display: flex;
    flex-direction: column;
    justify-content: center;
    align-items: center;
    flex: 1;
  }
  .title {
    font-size: 48px;
    font-family: var(--title-font);
    line-height: calc(48px * 2);
  }
  .pw-wrapper {
    margin-bottom: 24px;
  }
  .name {
    font-size: 16px;
    margin-bottom: 12px;
  }
  input {
    width: 30vw;
    min-width: 600px;
    line-height: 42px;
    border: solid 2px var(--btn-secondary);
    border-radius: 16px;
    background: none;
    padding-left: 20px;
  }
  input:active, :focus {
    outline: none; 
  }
  button {
    background: var(--btn-special);
    color: var(--text-card-color);
    margin: 20px;
    padding: 12px 32px;
    border: none;
    border-radius: 16px;
  }
  button:hover {
    cursor: pointer;
  }
  button:disabled {
    pointer-events: none;
    opacity: .6;
  }
</style>
