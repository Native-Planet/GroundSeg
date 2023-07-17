<script>
  import { fade } from 'svelte/transition'
  import { fadeIn, fadeOut, isPortrait } from '$lib/api'
</script>

<div in:fade={fadeIn} out:fade={fadeOut} class="wrapper">
  <div class="text">
    No connection to GroundSeg. Make sure GroundSeg is running on your device!
  </div>
</div>


<style>
  @font-face {
    font-family: Inter;
    src: url("/Inter-SemiBold.otf");
  }
  .wrapper {
    font-family: Inter;
    width: 100vw;
    transform: translate(-50%, -50%);
    background: #0404044d;
    position: absolute;
    top: 50%;
    left: 50%;
    text-align: center;
    backdrop-filter: blur(10px);
    -moz-backdrop-filter: blur(10px);
    -o-backdrop-filter: blur(10px);
    -webkit-backdrop-filter: blur(10px);
  }
  .text {
    color: #ffffff;
    font-family: inherit;
    padding: 60px;
  }
</style>
