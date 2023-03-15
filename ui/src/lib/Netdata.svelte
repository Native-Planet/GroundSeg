<script>
  import { api } from '$lib/api'
  import Fa from 'svelte-fa'
  import { faArrowUpRightFromSquare } from '@fortawesome/free-solid-svg-icons'
  import { faCircleQuestion } from '@fortawesome/free-regular-svg-icons'
  import PrimaryButton from '$lib/PrimaryButton.svelte'

  export let link
  let textToggle = false

  const handleTextToggle = () => {
    textToggle = !textToggle
  }

</script>

<div class="netdata">
  <div class="title-wrapper">
    <div class="title">Netdata

      <!-- Info button -->
      <button class="question-mark" on:click={handleTextToggle} >
        <Fa icon={faCircleQuestion} size="1.2x" />
      </button>
    </div>

    <!-- Info text -->
    <div class="button-wrapper">
      <a href={link} target="_blank">
        <Fa icon={faArrowUpRightFromSquare} size="1.2x" />
      </a>
    </div>
  </div>
  {#if textToggle}
    <div class="netdata-info">
      A detailed system overview of your device.
    </div>
  {/if}
</div>

<style>
  .netdata {
    background: #0000001d;
    padding: 20px 30px;
    border-radius: 8px;
    font-size: 18px;
  }
  .title-wrapper {
    display: flex;
    align-items: center;
  }
  .question-mark {
    color: inherit;
    cursor: pointer;
  }
  .title {
    font-size: 18px;
    flex: 1;
  }
  .button-wrapper {
    border-radius: 8px;
    padding: 2px;
  }
  a {
    margin: auto;
    margin-left: 16px;
    opacity: .8;
    color: inherit;
  }
  .netdata-info {
    font-size: 12px;
    margin-top: 12px;
  }
</style>
