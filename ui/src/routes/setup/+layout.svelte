<script>
  import { wide } from '$lib/stores/display'
  import { connect, connected } from '$lib/stores/websocket'
  import { structure } from '$lib/stores/data'

  let steps = [0,1,2]
  $: page = ($structure?.page) || 0
</script>
<div class="container {$wide ? "wide" : "slim"}">
  <slot />
</div>
<div class="steps">
  {#each steps as step}
    <div class="step" class:highlight={step <= page}></div>
  {/each}
</div>

<style>
  .container {
    background-color: var(--bg-base);
    color: var(--text-color);
    margin: auto;
    border-radius: 16px;
    position: relative;
    display: flex;
    flex-direction: column;
    justify-content: center;
    align-items: center;
    padding: 64px;
  }
  .wide {
    width: calc(992px - 128px);
    min-height: calc(100vh - 368px);
    margin-top: 64px;
    max-width: 98vw;
  }
  .slim {
    width: 100vw;
  }
  .steps {
    margin-top: 32px;
    display: flex;
    justify-content: center;
    gap: 8px;
  }
  .step {
    border-radius: 8px;
    width: 32px;
    height: 32px;
    background: #DDE3DF;
  }
  .highlight {
    background: #D7DB0F;
  }
</style>
