<script>
  import { onMount } from 'svelte'
  //import { getPatpArray } from '$lib/stores/patp'
  //import { renderPy } from '$lib/stores/websocket'
  import { sigil, stringRenderer } from '@tlon/sigil-js'
  import { parse, stringify } from 'svgson'
  export let patp
  export let size
  export let rad
  export let percent
  export let moon = false
  export let padding = "0px"

  let bg = '#000000';

  const parser = new DOMParser();
  const n = Math.floor(Math.random() * Math.pow(2, 32))

  const buildSVG = (p,bg) => {
    let svg;
    if (patp.length < 14) {
      return sigil({
        patp: p,
        margin: false,
        renderer: myRenderer,
        colors: [bg,'white'],
      })
    }
    if (patp.length < 28) {
      moon = true
      parent = patp.slice(-13)
      return sigil({
        patp: parent,
        margin: false,
        renderer: myRenderer,
        colors: [bg,'white']
    })}
    return "comet"
  }

  const myRenderer = e => {
    e.children[0]['attributes']['fill'] = "none"
    return stringify(e)
  }

  const renderSVG = (id,s) => {
    if (s != "comet") {
      var doc = new DOMParser().parseFromString(s, 'application/xml');
      var nid = id + "-" + patp + "-" + n
      var el = document.getElementById(nid)
      el.appendChild(el.ownerDocument.importNode(doc.documentElement, true))
    }
  }

  onMount(()=> {
    let root = getComputedStyle(document.documentElement);
    bg = root.getPropertyValue('--bg-card');
    renderSVG('sig', buildSVG(patp,bg))
  })

</script>

<div 
  class="wrapper"
  style="
    height:{size};
    width:{size};
    border-radius:{rad};
    padding: {padding};
    background: {
      percent == 0 
      ? "transparent"
      : "linear-gradient(to top, " + bg + " " + percent + "%, transparent " + (100 - percent) + "%;"
    }
    padding:{padding};">
  <!--background: linear-gradient(to top, {bg} {percent}%, transparent {100 - percent}%);-->

  <div
    class="sigil"
    id='sig-{patp}-{n}'
    style="
      width:{size};
      height:{size};
      ">
  <!--
  {#if moon}
    <div class="moon">moon</div>
  {/if}
  -->
  </div>
</div>

<style>
  .wrapper {
    overflow: hidden;
    position: relative;
  }
  .sigil {
    background-image: url("/comet.svg");
    background-size: contain;
  }

  /*
  .moon {
    border-radius: 4px;
    background: #040404e0;
    color: var(--text-card-color);
    position: absolute;
    bottom: 0;
    left: 0;
    right: 0;
    font-size: 14px;
    line-height: 28px;
    padding: 1px 3px 3px 3px;
    text-align: center;
  }
  */
</style>
