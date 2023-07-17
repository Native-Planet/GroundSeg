<script>
  import { createEventDispatcher } from 'svelte'

  export let value, listOptions

  let showList = false

  const dispatch = createEventDispatcher()

  const toggleList = () => showList = !showList

  const handleChange = e => {
    toggleList()
    dispatch('change',e)
  }
</script>

<div class="main-wrapper">
  <div class="main-display" on:click={toggleList} >{value < 10 ? "0" : ""}{value}</div>
  {#if showList}
    <div class="time-list">
      {#each listOptions as o}
        <div class="option" on:click={handleChange(o)}>{o < 10 ? "0" : ""}{o}</div>
      {/each}
    </div>
  {/if}
</div>
    <!-- hour selector --
    <Listbox value={selectedHour} on:change={e => {selectedHour = e.detail; console.log(selectedHour)}}>
      <ListboxOptions class="time-list">
        {#each hours as hour}
          <ListboxOption class="time-option" value={hour}>
            {hour < 10 ? "0" : ""}{hour}
          </ListboxOption>
        {/each}
      </ListboxOptions>
      <ListboxButton class="time-selector">{selectedHour < 10 ? "0" : ""}{selectedHour}</ListboxButton>
    </Listbox -->



<style>
  .main-wrapper {
    position: relative;
  }
  .main-display {
    padding: none;
    font-size: 12px;
    font-family: inherit;
    color: inherit;
    background: #FFFFFF4D;
    border-radius: 4px;
    padding: 2px 6px 2px 6px;
    cursor: pointer;
  }
	.time-list::-webkit-scrollbar {display: none;}
  .time-list {
    top: 24px;
    font-size: 12px;
    text-align: center;
    background: #040404;
    position: absolute;
    max-height: 64px;
    -ms-overflow-style: none;
		scrollbar-width: none;
    overflow: scroll;
    list-style-type: none;
    cursor: pointer;
    border-radius: 4px;
  }
  .option {
    padding: 2px 6px 2px 6px;
  }
</style>
