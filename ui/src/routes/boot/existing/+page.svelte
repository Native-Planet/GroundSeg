<script>
  import { existingID } from '$lib/components'
  import PrimaryButton from '$lib/PrimaryButton.svelte'
  import LinkButton from '$lib/LinkButton.svelte'

  let warningCheck = false
  let fileTypes = ['.zip','.tar','.tar.gz','.tgz']
  let allowCancel = true


  const hideCancel = () => allowCancel = false

</script>

<svelte:component this={existingID.logo} />
<div class="pier">
  {#if warningCheck}
    <div class="title">Upload a pier folder</div>
    <div class="subtitle">
      <div>Accepted Extensions:</div>
      {#each fileTypes as f}
        <div class="file-type">{f}</div>
      {/each}
    </div>
    <svelte:component this={existingID.dropzone} on:full={hideCancel}/>
    {#if allowCancel}
      <LinkButton
        top=24
        left={false}
        text="Cancel"
        src="/"
        disabled={false}
      />
    {/if}
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
    <a class="learn" href="/boot/existing">Learn more</a>
    <div class="buttons">
      <LinkButton
        text="Back"
        src="/"
        disabled={false}
      />
      <PrimaryButton
          on:click={()=>warningCheck = !warningCheck}
          standard="I understand"
          left={false}
      />
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
    max-width: calc(100vw - 40px);
  }

  .title {
    font-size: 1.3em;
    font-weight: 700;
    padding-bottom: 6px;
    text-align: left;
  }
  .subtitle {
    font-weight: 700;
    font-size: .9em;
    padding-bottom: 24px;
    display: flex;
    margin-top: 12px;
    gap: 6px;
    align-items: center;
  }
  .file-type {
    background: #000;
    font-size: 10px;
    padding: 4px 8px 4px 8px;
    border-radius: 6px;
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
</style>
