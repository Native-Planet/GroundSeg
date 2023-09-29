<script>
  import { createEventDispatcher } from 'svelte'
  import Fa from 'svelte-fa'
  import { faCheck } from '@fortawesome/free-solid-svg-icons'

  export let name = ''
  export let checkAll = false
  export let submitting = false
  export const forceSet = b => {
    check = b
    dispatch('update', {name:name,check:check})
  }

  let check = false

  const dispatch = createEventDispatcher()

  const handleCheck = () => {
    check = !check
    dispatch('update',{name:name,check:check})
  }

</script>

<!-- Checkbox -->
<div class="check-wrapper" class:freeze={submitting}>
  <div class="checkbox" class:highlight={check} on:click={handleCheck}>
    {#if check}
      <Fa icon={faCheck} size="1x"/>
    {/if}
  </div>
  <span class="patp" on:click={handleCheck}>
    {
      !checkAll ? '~' + name
      : check ? 'Unselect all'
      : 'Select all'
    }
  </span>
</div>

<style>
  .check-wrapper {
    flex: 1 0 calc(50% - 12px);
    font-family: var(--regular-font);
    font-size: 12px;
    display: flex;
    gap: 8px;
    align-items: center;
  }
  .checkbox {
    height: 16px;
    width: 14px;
    border: 1px solid var(--btn-secondary);
    border-radius: 6px;
    color: var(--text-card-color);
    padding-left: 2px;
  }
  span {
    font-size: 12px;
    cursor: pointer;
    user-select: none;
  }
  .highlight {
    background: var(--btn-secondary);
  }
  .freeze {
    opacity: .6;
    pointer-events: none;
  }
</style>
