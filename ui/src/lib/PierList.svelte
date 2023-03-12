<script>
	import { onMount } from 'svelte'
  import { scale } from 'svelte/transition'
	import { page } from '$app/stores'

	import { api } from '$lib/api'
	import Sigil from '$lib/Sigil.svelte'

  import Fa from 'svelte-fa'
  import { faArrowUpRightFromSquare } from '@fortawesome/free-solid-svg-icons'

  export let u
	let inView = false
  let code = null
  let count = 1

  const getUrbitCode = () => {
    if (inView && ($page.url.pathname == "/")) {
      if (u.running) {
        fetch($api + '/urbit?urbit_id=' + u.name, {
          method: 'POST',
          credentials: "include",
          headers: {'Content-Type': 'application/json'},
          body: JSON.stringify({'app':'pier','data':'+code'})
        })
        .then(r => r.json())
        .then(d => {
          code = d
          if (d.length == 27) {
            setTimeout(getUrbitCode, 1800000)
          } else {
            let time = 1000
            setTimeout(getUrbitCode, time * count)
            if (count < 5) {
              count = ++count
            }
          }
        })
      } else {
        setTimeout(getUrbitCode, 1000)
      }
  }}

	onMount(()=> {
		inView = true
    getUrbitCode()
	})

</script>
{#if inView}
  <div class="pier" in:scale={{duration:120, delay: 300}}>
    <Sigil patp={u.name} size="60px" rad="8px" />
    <a class="info" href={u.name}>
      <div class="patp">{u.name}</div>
      <div class="status">
        ({u.remote ? "Remote" : "Local"})
        {
        !u.running ? 'Stopped'
        : code == null ? 'Loading...'
        : code.length != 27 ? 'Booting'
        : code.length == 27 ? 'Running'
        : 'Loading...'
        }
      </div>
    </a>
    <a class="ext" href={u.running ? u.url : ""} target={u.running ? "_blank" : ""}>
      <Fa icon={faArrowUpRightFromSquare} size="1.2x" />
    </a>
  </div>
{/if}
<style>
	a { color: inherit; }
  .ext:hover {
    opacity: .6;
  }
  .pier {
    padding: 6px 20px 6px 20px;
    display: flex;
    align-items: center;
    flex-wrap: wrap;
  }
  .pier:hover {
    background: #00000080;
  }
  .info {
    display: flex;
    flex-direction: column;
    padding-left: 12px;
    flex: auto;
  }
  .patp {
    font-weight: 600;
    color: inherit;
  }
  .status {
    opacity: .8;
    font-weight: 400;
    font-size: .8em;
  }
</style>
