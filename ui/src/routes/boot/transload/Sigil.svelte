<script>
  import { afterUpdate } from 'svelte';
  import { sigRemove, checkPatp } from '$lib/stores/patp';
  import { sigil, stringRenderer } from '@tlon/sigil-js'
  export let name
  export let swap = false
  export let reverse = false
  export let coverage = 100
  export let moonbar = true

  $: noSig = sigRemove(name)
  $: validPatp = checkPatp(noSig)
  $: isMoon = (noSig.length == 27) || (noSig.length == 20)
  $: isPlanet = (noSig.length == 13)
  $: isStar = (noSig.length == 6)
  $: isGalaxy = (noSig.length == 3)

  let displayed = ""

  afterUpdate(()=> {
    if (validPatp && (isMoon || isPlanet || isStar || isGalaxy)) {
      let root = getComputedStyle(document.documentElement);
      let bg = root.getPropertyValue('--bg-modal');
      let fg = root.getPropertyValue('--text-color');
      if (swap) {
        let tmp = bg
        bg = fg
        fg = tmp
      }
      let patp = noSig
      if (isMoon) {
        patp = patp.slice(-13)
      } 
      displayed = sigil({
        patp: patp,
        renderer: stringRenderer,
        size: 120,
        colors: [bg, fg],
      })
    } else {
        displayed = ""
      }
  })

</script>
<div
  class="sigil"
  style="height: {reverse ? 128 : 128 - (128*coverage/100)}px;"
  >
    {@html displayed}
    {#if isMoon}<div class:moonbar={moonbar}>moon</div>{/if}
  </div>
<style>
  .sigil {
    position: relative;
    width: 128px;
    background: var(--bg-modal);
    overflow: hidden;
    transition: height 1000ms;
  }
  .moonbar {
    position: absolute;
    background: #5C7060BF;
    height: 24px;
    line-height: 24px;
    font-size: 9px;
    font-weight: 800;
    bottom: 0;
    right: 0;
    padding: 0 12px;
    text-align: center;
    color: var(--text-card-color);
    border-radius: 16px 0 0 0;
  }
</style>
