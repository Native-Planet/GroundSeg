<script>
  import { afterUpdate, createEventDispatcher, onMount } from 'svelte';
  import { checkPatp } from '../../../lib/stores/patp';

  let key = '', patp = '', viewKey = false, error = false, files

  $: if (files) {
    handleKey(files[0])
  }
  const dispatch = createEventDispatcher()

  let reader;
  onMount(()=>reader = new FileReader());

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
    const [_matched, name, extension] =
      file.name.match(/(\D*)(?:-\d*)?\.([^.]*)$/) || [];
    if (extension == 'key') {
      if (checkPatp(name)) {
        patp = name;
      }
      reader.readAsText(file)
      reader.onload = event => key = event.target.result;
    } else {
      error = true;
      setTimeout(()=> error = false, 1000);
    }
  }

  afterUpdate(()=> {
    dispatch("changeKey", key);
    dispatch("changePatp", patp);
  });
</script>

<div class="pass-wrapper">
  <input 
    spellcheck="false"
    id="key"
    type="password"
    placeholder={error ? "Not valid key file" : "Paste key or drop a keyfile"}
    bind:value={key}
    on:dragover={handleDragOver}
    on:drop={handleDrop}
  />
  <div class="upload-icon">
    <input type="file" bind:files >
    Choose
  </div>
  <!--
  <img on:click={toggleViewKey} src="/eye-{viewKey ? "closed" : "open"}.svg" alt="eye" />
  -->
</div>

<style>
  .pass-wrapper {
    display: flex;
  }
  input:focus {
    outline: none;
  }
  input {
    flex: 1;
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    letter-spacing: -1.44px;
    border-radius: 16px;
    padding: 10px 22px 12px 22px;
    width: calc(100% - 48px);
    border: 2px solid var(--Gray-400, #5C7060);
    color: var(--text-color);
    background: var(--bg-base);
  }
  input::placeholder {
    color: var(--Gray-200, #ABBAAE);
  }
  ::-moz-placeholder {
    opacity: .6;
  }
  ::-webkit-input-placeholder {
    opacity: .6;
  }
  .upload-icon {
    position: relative;
    margin-left: 16px;
    cursor: pointer;
    color: #FFF;
    text-align: center;
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    letter-spacing: -1.44px;
    border-radius: 16px;
    background: var(--Gray-400, #5C7060);
    padding: 12px 48px;
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
    filter: invert(100%);
  }
  input[type="file"]::before {
    content: "Choose File";
    display: inline-block;
    cursor: pointer;
  }
</style>
