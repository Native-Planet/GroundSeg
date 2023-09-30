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
    color: #000;
    leading-trim: both;
    text-edge: cap;
    font-family: var(--title-font);
    font-size: 48px;
    font-style: normal;
    font-weight: 700;
    line-height: normal;
    letter-spacing: -1.92px;
    margin-bottom: 55px;
  }
  .pw-wrapper {
    margin-bottom: 32px;
  }
  .name {
    color: var(--NP_Black, #313933);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    line-height: normal;
    letter-spacing: -1.44px;
    margin-bottom: 16px;
  }
  input {
    min-width: 600px;
    line-height: 65px;
    border: solid 2px var(--btn-secondary);
    border-radius: 16px;
    background: none;
    padding-left: 20px;
    color: var(--NP_Black, #313933);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    letter-spacing: -1.44px;
  }
  input:active, :focus {
    outline: none; 
  }
  button {
    margin-top: 55px;
    font-family: var(--regular-font);
    background: var(--btn-special);
    color: var(--text-card-color);
    height: 65px;
    font-size: 12px;
    border-radius: 16px;
    cursor: pointer;
    justify-content: center;
    align-items: center;
    padding: 0 48px;
    cursor: pointer;

    color: #FFF;
    text-align: center;
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    line-height: 32px; /* 133.333% */
    letter-spacing: -1.44px;
  }
  button:hover {
    cursor: pointer;
  }
  button:disabled {
    pointer-events: none;
    opacity: .6;
  }
</style>
