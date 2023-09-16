<script>
  import { afterUpdate } from 'svelte';
  import { sigRemove, checkPatp } from '$lib/stores/patp';
  import { sigil, stringRenderer } from '@tlon/sigil-js'
  export let name
  export let modal = false

  $: noSig = sigRemove(name)
  $: validPatp = checkPatp(noSig)
  $: isMoon = (noSig.length == 27) || (noSig.length == 20) || false
  $: isPlanet = (noSig.length == 13)
  $: isStar = (noSig.length == 6)
  $: isGalaxy = (noSig.length == 3)

  let displayed = ""

  afterUpdate(()=> {
    if (validPatp && (isMoon || isPlanet || isStar || isGalaxy)) {
      let root = getComputedStyle(document.documentElement);
      let bg
      let fg
      if (modal) {
        bg = root.getPropertyValue('--bg-modal');
        fg = root.getPropertyValue('--text-color');
      } else {
        bg = root.getPropertyValue('--fg-card');
        fg = root.getPropertyValue('--text-card-color');
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
<div class="{modal ? "modal" : "sigil"}">{@html displayed}</div>
<style>
  .sigil {
    width: 64px;
    height: 64px;
    background: var(--btn-secondary);
    overflow: hidden;
    margin-left: 34px;
    margin-top: 38px;
  }
  .modal {
    margin-top: 48px;
    width: 120px;
    height: 120px;
    overflow: hidden;
    border: solid 4px var(--text-color);
    border-radius: 16px;
  }
</style>
