<script>
  import { structure } from '$lib/stores/websocket'
  export let wide
  import Fa from 'svelte-fa'
  import { faArrowRight } from '@fortawesome/free-solid-svg-icons';
</script>

<div class="container {wide ? "wide" : "slim"}">
    <div class="title">StarTram</div>
    <div class="wrapper">
      <div class="subs">
        <div class="sub">Subscription ends (date here)</div>
        <div class="sub">Active region: (region)</div>
        <div class="sub">Connection Status: (status)</div>
      </div>
      <button class="edit-btn">Edit <Fa icon={faArrowRight} size="0.8x" /></button>
    </div>
</div>

<style>
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
  .wrapper {
    display: flex;
    align-items: center;
  }
  .subs {
    flex: 1;
    display: flex;
    gap: 8px;
  }
  .sub {
    color: var(--text-color);
    text-decoration: none;
    font-size: 14px;
  }
  .edit-btn {
    font-family: var(--regular-font);
    background: var(--btn-secondary);
    color: var(--text-card-color);
    height: 42px;
    font-size: 14px;
    width: 120px;
    border-radius: 16px;
    cursor: pointer;
    display: flex;
    gap: 6px;
    justify-content: center;
    align-items: center;
  }
</style>
