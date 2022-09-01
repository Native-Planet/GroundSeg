<script>
  import { onMount } from 'svelte'
  import { url } from '/src/Scripts/server'
  import Dropzone from "dropzone"

  onMount(()=> {
    let myDropzone = new Dropzone("#dropper", {
      paramName: "file", // The name that will be used to transfer the file
      acceptedFiles: '.zip, .tar, .tgz, .gz',
      chunking: true,
      forceChunking: true,
      url: url + '/upload/pier',
      maxFilesize: 11000000, // megabytes
      chunkSize: 50000000 // bytes
  })})

</script>

<div id="dropper" class="drop">
  <div class="text">Drop pier here to upload</div>
</div>

<style>
  .drop {
    position: relative;
    height: 120px;
    text-align: center;
    border: 1px solid #ffffff80;
  }
  .text {
    position: absolute;
    top: calc(50% - .5em);
    width: 100%;
    margin: auto;
  }
</style>
