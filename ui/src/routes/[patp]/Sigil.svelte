<script>
  import { afterUpdate } from 'svelte';
  import { sigRemove, checkPatp } from '$lib/stores/patp';
  import { sigil, stringRenderer } from '@tlon/sigil-js'
  export let name

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
      let bg = root.getPropertyValue('--fg-card');
      let fg = root.getPropertyValue('--text-card-color');
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
<div class="sigil">{@html displayed}</div>
<style>
  .sigil {
    width: 64px;
    height: 64px;
    background: var(--btn-secondary);
    overflow: hidden;
    margin-left: 34px;
    margin-top: 38px;
  }
</style>
