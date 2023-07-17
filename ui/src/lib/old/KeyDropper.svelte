<script>
  import Dropzone from "dropzone"
  import { afterUpdate, createEventDispatcher } from "svelte"

  import Fa from 'svelte-fa'
  import { faFileArrowUp } from '@fortawesome/free-solid-svg-icons'

  let key = '', viewKey = false, error = false, files

  $: if (files) {
    handleKey(files[0])
  }
  const dispatch = createEventDispatcher()

  const reader = new FileReader();

  const toggleViewKey = () => {
    viewKey = !viewKey
    document.querySelector('#key').type = viewKey ? 'text' : 'password'
  }

  const handleDragOver = event => {
    event.preventDefault();
  }

  const handleDrop = event => {
    event.preventDefault();
    handleKey(event.dataTransfer.files[0])
  }

  const handleKey = file => {
    if (file.name.split('.').splice(-1)[0] == 'key') {
      reader.readAsText(file)
      reader.onload = event =>  key = event.target.result
    } else {
      error = true
      setTimeout(()=> error = false, 1000)
    }
  }

  afterUpdate(()=> {
    dispatch("change", key)
  })

</script>

<div class="pass-wrapper">
  <input 
    spellcheck="false"
    id="key"
    type="password"
    placeholder={error ? "not valid key file" : "paste key or drop a keyfile"}
    bind:value={key}
    on:dragover={handleDragOver}
    on:drop={handleDrop}
  />
  <div class="upload-icon">
    <input type="file" bind:files >
    <Fa icon={faFileArrowUp} size="1.2x" />
  </div>
  <img on:click={toggleViewKey} src="/eye-{viewKey ? "closed" : "open"}.svg" alt="eye" />
</div>

<style>
  .pass-wrapper {
    display: flex;
  }
  input {
    flex: 1;
    padding: 8px;
    font-size: 12px;
    color: inherit;
    font-weight: 700;
    background: #ffffff4d;
    outline: none;
    border: none;
    border-radius: 6px;
  }
  ::-moz-placeholder {
    color: white;
  }
  ::-webkit-input-placeholder {
    color: white;
  }
  .upload-icon {
    position: relative;
    margin: auto;
    padding: 0 12px 0 12px;
    cursor: pointer;
  }
  .upload-icon > input {
    cursor: pointer;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    opacity: 0;
    position: absolute;
    overflow: hidden;
  }
  img {
    cursor: pointer;
  }
  input[type="file"]::before {
    content: "Choose File";
    display: inline-block;
    cursor: pointer;
  }
</style>
