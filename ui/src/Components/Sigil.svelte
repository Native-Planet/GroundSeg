<script>
  import { onMount } from 'svelte'
  import { sigil, stringRenderer } from '@tlon/sigil-js'
  export let patp, size, rad, moon = false

  const parser = new DOMParser();
  const n = Math.floor(Math.random() * Math.pow(2, 32))

  const sig = () => {
    if (patp.length < 14) {
      return sigil({
        patp: patp,
        renderer: stringRenderer,
        colors: ['#040404','white']
    })}

    if (patp.length < 28) {
      moon = true
      parent = patp.slice(-13)
      return sigil({
        patp: parent,
        renderer: stringRenderer,
        colors: ['#040404','white']
    })}

    return "comet"
  }

  const renderSVG = (id,s) => {
    if (s != "comet") {
      var doc = new DOMParser().parseFromString(s, 'application/xml');
      var nid = id + "-" + patp + "-" + n
      var el = document.getElementById(nid)
      el.appendChild(el.ownerDocument.importNode(doc.documentElement, true))
    }}

  onMount(()=> {
    renderSVG('sig', sig())
  })

</script>
<div
    id='sig-{patp}-{n}'
    style="
      width:{size};
      height:{size};
      border-radius:{rad}">
  {#if moon}
    <div class="moon">moon</div>
  {/if}
</div>

<style>
  div {
    overflow:hidden;
    background-color: #040404;
    background-image: url("/comet.svg");
    background-size: contain;
    position: relative;
  }

  .moon {
    border-radius: 4px;
    background: #040404e0;
    position: absolute;
    bottom: 0;
    left: 0;
    right: 0;
    font-size: 9px;
    line-height: 14px;
    padding: 1px 3px 3px 3px;
    text-align: center;
  }
</style>
