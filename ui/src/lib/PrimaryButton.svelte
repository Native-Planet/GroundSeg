<script>
	import { createEventDispatcher } from 'svelte';

  export let standard=null,
    success=null,
    failure=null,
    loading=null,
    status='standard', 
    top=0, left=true, bottom=0

	const dispatch = createEventDispatcher();

  const handleClick = () => dispatch('click', null)

</script>

<div style="margin-bottom:{bottom}px;margin-top:{top}px;margin-{left ? "right" : "left"}:auto;">
{#if status == 'standard'}
  <button on:click={handleClick}>
    {standard}
  </button>
{/if}

{#if status == 'loading'}
  <div class="loading">
    {loading}
  </div>
{/if}

{#if status == 'success'}
  <div class="success">{success}</div>
{/if}

{#if status == 'failure'}
  <div class="failure">{failure}</div>
{/if}

{#if status == 'disabled'}
  <button class="disabled">
    {standard}
  </button>
{/if}
</div>

<style>
  button {
    color: var(--text-color);
    background: var(--action-color);
    border: none;
    border-radius: 6px;
    padding: 8px 12px 8px 12px;
    min-width: 80px;
    line-height: 14px;
    font-size: 12px;
    cursor: pointer;
    font-family: inherit;
  }
  .disabled {
    opacity: .6;
    pointer-events: none;
  }
  .loading {
    font-size: 12px;
    line-height: 30px;
    animation: breathe 2s infinite;
  }
  .success {
    color: lime;
    font-size: 12px;
    line-height: 30px;
  }
  .failure {
    color: red;
    font-size: 12px;
    line-height: 30px;
  }
</style>
