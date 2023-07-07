<script>
  import { wide } from '$lib/stores/display'
  import { connect, structure, connected } from '$lib/stores/websocket'

  let steps = [0,1,2]
  $: page = ($structure?.system?.setup?.page) || 0
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
    gap: 20px;
    background-color: var(--bg-base);
    color: var(--text-color);
    margin: auto;
    border-radius: 16px;
    position: relative;
    display: flex;
    flex-direction: column;
    justify-content: center;
    align-items: center;
  }
  .wide {
    width: 992px;
    height: calc(100vh - 240px);
    margin-top: 120px;
    max-width: 98vw;
  }
  .slim {
    width: 100vw;
  }
  .steps {
    margin-top: 24px;
    display: flex;
    justify-content: center;
    gap: 4px;
  }
  .step {
    border-radius: 4px;
    width: 16px;
    height: 16px;
    background: #DDE3DF;
  }
  .highlight {
    background: #D7DB0F;
  }
</style>
