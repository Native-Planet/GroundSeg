<script>
  import { beginSetup } from '$lib/stores/websocket'
</script>

<div class="wrapper">
  <img src="/groundseg.svg" alt="groundseg" />
  <div class="title">GROUNDSEG</div>
  <div class="subtitle">Click "Begin" to begin to start your device setup</div>
  <button on:click={beginSetup} >Begin</button>
</div>

<style>
  .wrapper {
    display: flex;
    flex-direction: column;
    justify-content: center;
    align-items: center;
  }
  img {
    width: 48px;
  }
  .title {
    font-size: 48px;
    font-family: var(--title-font);
  }
  .subtitle {
    line-height: 42px;
    font-size: 18px;
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
