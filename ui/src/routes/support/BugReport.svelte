<script>
</script>

<div class="container">
  <div class="title">BUG REPORT</div>

  <div class="wrapper">
    <div class="left">
      <div class="contact">
        <div class="header">How should we contact you?</div>
        <input class="contact-input" placeholder="~sampel-palnet or example@email.com"/>
      </div>
      <div class="description">
        <div class="header">Describe your issue in detail</div>
        <input class="description-input" placeholder="eg. ~zod doesn't turn after update"/>
      </div>
    </div>

    <div class="right">
      <div class="info">
        <div class="header">Information sent to Native Planet</div>
        <ul>
          <li>GroundSeg logs</li>
          <li>StarTram information (if available)</li>
          <li>List of Docker containers on your device</li>
          <li>GroundSeg and Docker container configs (removed private information)</li>
        </ul>
      </div>
      <div class="logs">
        <div class="header">Send Pier Logs (optional)</div>
        <div class="check-flex">
          <div class="check-wrapper">
            <div class="checkbox"></div>
            <div class="patp">nattev-palnet</div>
          </div>
          <div class="check-wrapper">
            <div class="checkbox"></div>
            <div class="patp">nattev-palnet</div>
          </div>
          <div class="check-wrapper">
            <div class="checkbox"></div>
            <div class="patp">nattev-palnet</div>
          </div>
        </div>
      </div>
      <div class="buttons">
        <div class="spacer"></div>
        <button class="submit">Submit Report</button>
      </div>
    </div>
  </div>
</div>

<style>
  .container {
    margin: 0;
  }
  .title {
    margin-bottom: 12px;
  }
  .wrapper {
    display: flex;
    gap: 40px;
  }
  .left {
    flex: 1;
  }
  .header {
    font-family: var(--regular-font);
    font-size: 13px;
    line-height: 24px;
    margin-bottom: 4px;
    margin-top: 12px;
  }
  .right {
    flex: 1;
  }
  input {
    width: 100%;
    font-family: var(--regular-font);
    font-size: 12px;
    padding: 8px;
    border-radius: 8px;
    border: solid 2px var(--btn-secondary);
    background: var(--bg-modal);
    color: var(--text-color);
  }
  input:focus {
    outline: none;
  }
  input::placeholder {
    color: var(--text-color);
    opacity: .6;
  }
  .description-input {
    height: 200px;
  }
  .info {
    padding: 10px 20px;
    border-radius: 8px;
    background: var(--bg-card);
    color: var(--text-card-color);
    margin-bottom: 20px;
  }
  .logs {
    padding: 20px;
    padding-top: 0;
    border-radius: 8px;
    border: solid 2px var(--btn-secondary);
    background: var(--bg-modal);
  }
  .check-flex {
    display: flex;
    flex-wrap: wrap;
    gap: 12px;
    margin-top: 10px;
  }
  .check-wrapper {
    flex: 1 0 calc(50% - 12px);
    font-family: var(--title-font);
    font-size: 12px;
    display: flex;
    gap: 8px;
  }
  .checkbox {
    height: 16px;
    width: 16px;
    border: 1px solid var(--btn-secondary);
    border-radius: 6px;
  }
  ul {
    font-size: 12px;
    font-family: var(--regular-font);
  }
  li {
    padding: 2px;
  }
  .buttons {
    display: flex;
  }
  .spacer {
    flex:1;
  }
  .submit {
    line-height: 36px;
    font-size: 12px;
    font-family: var(--regular-font);
    background: var(--btn-primary);
    color: var(--text-card-color);
    margin-top: 20px;
    width: 240px;
    border-radius: 8px;
  }
  .submit:hover {
    cursor: pointer;
    background: var(--bg-card);
  }
</style>
