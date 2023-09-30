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
    color: #000;
    leading-trim: both;
    text-edge: cap;
    font-family: var(--title-font);
    font-size: 48px;
    font-style: normal;
    font-weight: 700;
    line-height: normal;
    letter-spacing: -1.92px;
  }
  .subtitle {
    color: var(--NP_Black, #313933);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    line-height: normal;
    letter-spacing: -1.44px;
    margin: 48px;
  }
  button {
    margin-top: 20px;
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
</style>
