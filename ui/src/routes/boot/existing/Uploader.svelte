<script>
  import { processFile } from '$lib/stores/uploader'

  // Initialize the file object
  let file;

  // get file size
  // TODO
  // get number of chunks. Chunk size = 25MB
  // TODO
  // stream first 25MB of the file
  // let data = TODO
  // get the hash of the byte stream
  // let hash = TODO
  // call this chunk=n where n is the chunk number
  // TODO
  // create an index db entry with {"chunk":chunk,"hash":hash,"data":data}
  // and then continue with the next 25MB until everything is complete



  function onFileChange(event) {
    file = event.target.files[0];
  }

  function onSubmit() {
    if (file) {
      processFile(file);
    } else {
      alert('Please select a file');
    }
  }
</script>

<form on:submit|preventDefault={onSubmit}>
  <label for="file">Choose a file:</label>
  <input type="file" id="file" name="file" on:change={onFileChange} />
  <input type="submit" value="Submit">
</form>

<style>
  input {
    width: 30vw;
    min-width: 600px;
    line-height: 42px;
    border: solid 2px var(--btn-secondary);
    border-radius: 16px;
    background: none;
    padding-left: 20px;
  }
  input:active, :focus {
    outline: none; 
  }
</style>
