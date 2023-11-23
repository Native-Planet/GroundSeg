<script>
  import Fa from 'svelte-fa'
  import { faArrowRight } from '@fortawesome/free-solid-svg-icons';

  export let memUsage = 0
  export let diskUsage = 0
  export let loom = 2048

  $: memPercent = ((memUsage / (1024 * 1024))/loom * 100).toFixed(1)
  $: memBlocks = Math.ceil(memPercent / 10)
  $: diskMB = (diskUsage / (1024 * 1024)).toFixed(2)
</script>

<div class="item">
  <div class="label">S</div>
  <div class="label">{diskMB} MB</div>
</div>
<div class="item">
  <div class="label">R</div>
  <div class="rects">
    {#each Array.from({ length: 10 }, (_, i) => i) as n}
      <div class="rect" class:active={memBlocks > n}></div>
    {/each}
  </div>
</div>

<style>
  .item {
    display: flex;
    gap: 4px;
  }
  .label {
    color: var(--text-card-color,#F8F8F6);
    leading-trim: both;
    text-edge: cap;
    font-family: var(--title-font);
    font-size: 12px;
    font-style: normal;
    font-weight: 700;
    line-height: normal;
    letter-spacing: -0.72px;
    text-transform: uppercase;
  }
  .rects {
    display: flex;
    gap: 2px;
    align-items: end;
  }
  .rect {
    background: var(--Gray-300, #8FA393);
    border-radius: 1px;
    width: 8px;
    height: 8px;
    margin-bottom: 2px;
  }
  .active {
    background: var(--text-card-color);
  }
</style>
