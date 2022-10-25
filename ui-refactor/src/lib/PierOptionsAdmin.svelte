<script>
  export let running
  let exportButtonText = 'Export Urbit Pier', deleteButtonText = 'Delete Urbit Pier'
  const exportUrbitPier = () => console.log("exported pier")
  const deleteUrbitPier = () => console.log("deleted pier") 

</script> 

<div class="option-title">Admin Actions</div>
<button class="export-pier" on:click={exportUrbitPier}>
  {exportButtonText}
</button>
<button class="delete-pier" on:click={deleteUrbitPier}>{deleteButtonText}</button>

<style>
  .option-title {
    font-size: 14px;
    color: inherit;
    margin-bottom: 4px;
  }

  .export-pier {
    color: orange;
    cursor: pointer;
  }

  .delete-pier {
    color: red;
    cursor: pointer;
  }
</style>
