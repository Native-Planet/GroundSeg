<script>
  import { api } from '$lib/api'
  import Fa from 'svelte-fa'
  import { faArrowUpRightFromSquare } from '@fortawesome/free-solid-svg-icons'
  import { faCircleQuestion } from '@fortawesome/free-regular-svg-icons'
  import Slider from '$lib/Slider.svelte'
  import PrimaryButton from '$lib/PrimaryButton.svelte'

  export let swapVal
  export let maxSwap

  let textToggle = false
  let hideSlider = true
  let curSwapVal = swapVal
  let defaultButtonStatus = "standard"
  let setButtonStatus = "standard"

  const handleTextToggle = () => {
    textToggle = !textToggle
  }

  const toggleSlider = () => {
    curSwapVal = swapVal
    hideSlider = !hideSlider
  }

  const setSwap = () => {
    setButtonStatus = 'loading'

    let module = 'swap'
	  fetch($api + '/system?module=' + module, {
			method: 'POST',
      credentials: 'include',
			headers: {'Content-Type': 'application/json'},
			body: JSON.stringify({'action': 'set','val': curSwapVal})
	  })
      .then(d => d.json())
      .then(res => {
        if (res === 200) {
          setButtonStatus = 'success'
          setTimeout(()=> {
            setButtonStatus = 'standard'
            hideSlider = true
          }, 3000)
        } else {
          setButtonStatus = 'failure'
          setTimeout(()=> setButtonStatus = 'standard', 3000)
        }
      })
      .catch(err => {
        console.log(err)
        setButtonStatus = 'failure'
        setTimeout(()=> setButtonStatus = 'standard', 3000)
      })
  }

  const setDefaultSwap = () => {
    curSwapVal = 16
    defaultButtonStatus = 'loading'

    let module = 'swap'
	  fetch($api + '/system?module=' + module, {
			method: 'POST',
      credentials: 'include',
			headers: {'Content-Type': 'application/json'},
			body: JSON.stringify({'action': 'set','val': curSwapVal})
	  })
      .then(d => d.json())
      .then(res => {
        if (res === 200) {
          defaultButtonStatus = 'success'
          setTimeout(()=> {
            defaultButtonStatus = 'standard'
            hideSlider = true
          }, 3000)
        } else {
          defaultButtonStatus = 'failure'
          setTimeout(()=> defaultButtonStatus = 'standard', 3000)
        }
      })
      .catch(err => {
        console.log(err)
        defaultButtonStatus = 'failure'
        setTimeout(()=> defaultButtonStatus = 'standard', 3000)
      })
  }

</script>

<div class="swap">
  <div class="title-wrapper">
    <div class="title">Swap Memory
      <!-- Info button -->
      <button class="question-mark" on:click={handleTextToggle} >
        <Fa icon={faCircleQuestion} size="1.2x" />
      </button>
    </div>
  </div>
  {#if textToggle}
    <!-- Info text -->
    <div class="netdata-info">
      A temporary storage area that helps your device manage tasks when its main memory (RAM) gets crowded.
    </div>
  {/if}
  <div class="swap-wrapper">
    {#if hideSlider}
      <div class="display-wrapper">
        <div class="swap-val">{swapVal} GB</div>
        <PrimaryButton 
          noMargin={true}
          status="standard"
          standard="Modify"
          on:click={toggleSlider}
          />
      </div>
    {:else}
      <Slider 
        on:change={(e) => curSwapVal = e.detail.value}
        min={1}
        max={maxSwap}
        initialValue={swapVal}
        value={curSwapVal}
      />
      <div class="button-wrapper">
        <PrimaryButton 
          status="standard"
          standard="Cancel"
          background="#ffffff4d"
          on:click={toggleSlider}
          />
        <PrimaryButton 
          status={(curSwapVal != 16) || (defaultButtonStatus != "standard") ? defaultButtonStatus : "disabled"}
          loading="resetting swap.."
          failure="something went wrong"
          success="swap reset!"
          standard="Set Default"
          on:click={setDefaultSwap}
          />
        <PrimaryButton 
          left={false}
          status={(swapVal != curSwapVal) || (setButtonStatus != "standard") ? setButtonStatus : "disabled"}
          loading="setting swap.."
          failure="something went wrong"
          success="swap set!"
          standard="Set Swap"
          on:click={setSwap}
          />
      </div>
    {/if}
  </div>
</div>

<style>
  .swap {
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
  .display-wrapper {
    display: flex;
    align-items: end;
    margin-top: 4px;
  }
  .swap-val {
    flex: 1;
    font-size: 14px;
  }
  .button-wrapper {
    display: flex;
    margin-top: 8px;
  }
  .netdata-info {
    font-size: 12px;
    margin-top: 6px;
  }
</style>
