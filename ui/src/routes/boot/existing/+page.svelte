<script>
  import Logo from '/src/Components/Buttons/Logo.svelte'
  import Dropzone from "svelte-file-dropzone";

  let files = {
    accepted: [],
    rejected: []
  };

  function handleFilesSelect(e) {
    const { acceptedFiles, fileRejections } = e.detail;
    files.accepted = [...files.accepted, ...acceptedFiles];
    files.rejected = [...files.rejected, ...fileRejections];
    if (files.accepted.length === 1) {
      console.log(files)
    /*
    const f = new FormData()
    f.append("patp", name)
    f.append("key", key)
    const r = fetch(url, {
      method: 'POST',
      body: f
    })
    */
    // handle redirect after success
    // .then(res => console.log(res))
    }
  }
  let warningCheck = false

</script>

<Logo />
<div class="pier">
  {#if warningCheck}
    <div class="title">Upload a pier folder</div>
    <div class="subtitle">Should be zip'd or tar'd. When finished please refresh page.</div>
    <!--
    <Dropzone on:drop={handleFilesSelect} accept="application/zip, application/gzip"/>
    -->
    <div id="dropper" class="drop">
      <div class="text">Drop files here to upload</div>
    </div>
    <a class="cancel-button" href="/">Cancel</a>
  {:else}
    <div class="warning-title">
      WARNING:
    </div>
    <div class="header">
      You have chosen to boot an existing Urbit ID...
    </div>
    <div class="warning">
      <div class="warning-content">
        <span>
          If your ID has already been booted on another device, you 
        </span>
        <strong>MUST</strong>
        <span>
          shut down your ship on the other device before uploading  your Pier File on the next page to avoid duplicating your ship.
        </span>
      </div>
      <div class="warning-content">
        <span>
          Once your ship is booted on your Native Planet device, proceed to delete the Pier File on your other hosting device or service.
        </span>
      </div>
      <div class="warning-content">
        <span>
          Neglecting to do so will cause technical and networking issues with your Urbit ID.
        </span>
      </div>
    </div>
    <a class="learn" href="">Learn more</a>
    <div class="buttons">
      <a href="/" class="back-button">Back</a>
      <button class="accept-button" on:click={()=> warningCheck = !warningCheck}>I Understand</button>
    </div>
  {/if}
</div>
<style>

  strong {
    opacity: 1;
  }

  .pier {
    display: flex;
    flex-direction: column;
    color: inherit;
    padding: 20px;
    width: 460px;
    max-width: calc(80vw - 40px);
  }

  .title {
    font-size: 1.3em;
    font-weight: 700;
    padding-bottom: 6px;
    text-align: left;
  }
  .subtitle {
    opacity: .7;
    font-weight: 700;
    font-size: .9em;
    padding-bottom: 24px;
  }
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
  .cancel-button {
    margin-top: 24px;
    padding: 12px 20px 12px 20px;
    color: inherit;
    text-align: center;
    font-weight: 500;
    border-radius: 8px;
    background: #FFFFFF4D; 
    font-size: 16px;
  }

  .warning-title {
    font-size: 12px;
    font-weight: 700;
    padding-bottom: 12px;
    text-align: left;
  }
  .header {
    font-size: 20px;
    font-weight: 700;
    padding-bottom: 14px;
    text-align: left;
  }
  .warning {
    color: inherit;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }
  .warning-content > span {
    font-size: 13px;
    opacity: .9;
  }
  .learn {
    padding-top: 18px;
    font-size: 12px;
    text-decoration-line: underline;
  }
  .buttons {
    display: flex;
    margin-top: 24px;
  }

  .back-button {
    padding: 12px 20px 12px 20px;
    color: inherit;
    text-align: center;
    font-weight: 500;
    border-radius: 8px;
    background: #FFFFFF4D; 
    font-size: 16px;
  }
  .accept-button {
    padding: 12px 20px 12px 20px;
    font-weight: 500;
    background: #28B8ED;
    color: inherit;
    border-radius: 8px;
    border: none;
    font-size: 16px;
    font-family: inherit;
    margin-left: auto;
    cursor: pointer;
  }

</style>
