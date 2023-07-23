<script>
  import { structure } from '$lib/stores/websocket'
  export let wide

  const modifyPassword = () => {
    console.log("modify")
  }
</script>

<div class="container {wide ? "wide" : "slim"}">
    <div class="title">PASSWORD</div>
    <div class="wrapper">
      <div class="modify">
        <div class="input-wrapper">
          <div class="label">Current Password</div>
          <input class="pwd" type="password" />
        </div>
        <div class="input-wrapper">
          <div class="label">New Password</div>
          <input class="pwd" type="password" />
        </div>
        <div class="input-wrapper">
          <div class="label">Confirm Password</div>
          <input class="pwd" type="password" />
        </div>
        <button class="btn" on:click={modifyPassword}>Modify Password</button>
      </div>
      <div class="spacer"></div>
      <div class="buttons">
        <button class="btn" on:click={modifyPassword}>Log Out Everywhere</button>
        <button class="btn" on:click={modifyPassword}>Log Out</button>
      </div>
    </div>
</div>

<style>
  .container {
    margin: auto;
  }
  .wrapper {
    display: flex;
    align-items: end;
  }
  .spacer {
    flex: 1;
  }
  .modify {
    display: flex;
    flex: 1;
    flex-direction: column;
    gap: 24px;
  }
  .pwd {
    width: calc(100% - 20px);
    font-family: var(--regular-font);
    color: var(--text-color);
    padding-left: 20px;
    border: 2px solid var(--btn-secondary);
    background-color: var(--bg-modal);
    border-radius: 12px;
    font-size: 16px;
    line-height: 36px;
  }
  .pwd:focus {
    outline: none;
  }
  .label {
    font-size: 12px;
    color: var(--text-color);
    margin-bottom: 8px;
  }
  .buttons {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }
  .btn {
    font-family: var(--regular-font);
    background: var(--btn-secondary);
    color: var(--text-card-color);
    height: 42px;
    font-size: 12px;
    border-radius: 16px;
    cursor: pointer;
    display: flex;
    gap: 6px;
    justify-content: center;
    align-items: center;
  }
  .btn:hover {
    background: var(--bg-card);
  }

  .wide {
    width: 992px;
    max-width: 98vw;
    padding: 56px;
  }
  .slim {
    width: 100vw;
  }
  .title {
    margin-bottom: 24px;
  }
  .links {
    display: flex;
    gap: 20px;
  }
  .link {
    text-decoration: none;
    color: var(--text-color);
    font-size: 12px;
  }
  .power-btn {
    background: var(--btn-secondary);
    color: var(--text-card-color);
    height: 42px;
    font-size: 14px;
    width: 200px;
    border-radius: 16px;
    cursor: pointer;
  }
</style>
