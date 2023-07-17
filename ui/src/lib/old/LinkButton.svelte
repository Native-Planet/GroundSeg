<script>
  import { createEventDispatcher } from 'svelte';

  export let text=null, src=null, top=0, left=true, disabled=true

</script>

<a style="margin{left ? "-right" : ""}:auto;margin-top:{top}px;"
  href={src}
  class:disabled={disabled}>
  {text}
</a>

<style>
  a {
    color: var(--text-color);
    padding: 8px 12px 8px 12px;
    min-width: 50px;
    line-height: 14px;
    font-size: 12px;
    cursor: pointer;
    font-family: inherit;
    text-align: center;
  }
  .disabled {
    opacity: .6;
    pointer-events: none;
  }
</style>
