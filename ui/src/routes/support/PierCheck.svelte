<script>
  import { createEventDispatcher } from 'svelte'
  import Fa from 'svelte-fa'
  import { faCheck } from '@fortawesome/free-solid-svg-icons'

  export let name = ''
  export let checkAll = false
  export let submitting = false
  export let checked = false

  const dispatch = createEventDispatcher()

  const handleCheck = () => {
    if (submitting) return
    dispatch('update',{name:name,check:!checked})
  }

</script>

<!-- Checkbox -->
<button type="button" class="check-wrapper" class:freeze={submitting} class:active={checked} disabled={submitting} aria-pressed={checked} on:click={handleCheck}>
  <span class="checkbox" class:highlight={checked}>
    {#if checked}
      <Fa icon={faCheck} size="1x"/>
    {/if}
  </span>
  <span class="patp">
    {
      !checkAll ? '~' + name
      : checked ? 'Unselect all'
      : 'Select all'
    }
  </span>
</button>

<style>
  .check-wrapper {
    flex: 1 0 calc(50% - 12px);
    font-family: var(--regular-font);
    font-size: 12px;
    display: flex;
    gap: 8px;
    align-items: center;
    min-height: 34px;
    padding: 8px;
    border: 1px solid var(--btn-secondary);
    border-radius: 8px;
    background: transparent;
    color: var(--text-color);
    text-align: left;
    cursor: pointer;
  }
  .checkbox {
    height: 16px;
    width: 14px;
    flex: 0 0 14px;
    border: 1px solid var(--btn-secondary);
    border-radius: 6px;
    color: var(--text-card-color);
    padding-left: 2px;
  }
  .active {
    background: var(--bg-card);
    color: var(--text-card-color);
  }
  span {
    font-size: 12px;
    user-select: none;
  }
  .patp {
    min-width: 0;
    overflow-wrap: anywhere;
  }
  .highlight {
    background: var(--btn-secondary);
  }
  .freeze {
    opacity: .6;
    pointer-events: none;
  }
</style>
