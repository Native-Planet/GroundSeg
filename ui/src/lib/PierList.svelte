<script>

	import { onMount } from 'svelte'
  import { scale } from 'svelte/transition'

	import { urbits, codes, api } from '$lib/api'
	import Sigil from '$lib/Sigil.svelte'

  import Fa from 'svelte-fa'
  import { faGear } from '@fortawesome/free-solid-svg-icons'

	let inView = false

  const checkStatus = (n,r) => {
	  fetch($api + '/urbit?urbit_id=' + n, {
		  method: 'POST',
		  headers: {'Content-Type': 'application/json'},
		  body: JSON.stringify({'app':'pier','data':'+code'})
	  })
      .then(r => r.json())
      .then(d => {
        codes.update(c => {
          c[n] = d
          return c
        })
        return d
      })
      .then(v => {
        if ((v.length != 27) && inView && r) {
          checkStatus(n)
        }
      })
  }

	onMount(()=> {
		inView = !inView
    for (let i = 0; i < $urbits.length; i++) {
      checkStatus($urbits[i].name, $urbits[i].running)
    }
	})

</script>
	<div class="wrapper">

    {#if $urbits.length == 0}

      <div class="welcome" in:scale={{duration:120, delay: 300}}>
        Welcome to GroundSeg.
      </div>
      <div class="welcome" in:scale={{duration:120, delay: 300}}>
        From here you can boot and manage multiple Urbit IDs.
      </div>
      <div class="welcome" in:scale={{duration:120, delay: 300}}>
        Select one of the options below to get started.
      </div>

    {:else} 

 		{#each $urbits as u, i}
			{#if inView}
		 		<div class="pier" in:scale={{duration:120, delay: 300}}>
			   	<Sigil patp={u.name} size="60px" rad="8px" />
					<a class="info"
    	  		href={u.running ? u.urbitUrl : ""}
		      	target={u.running ? "_blank" : ""}>
	  		    <div class="patp">{u.name}</div>
            <div class="status">
              {
              !u.running ? 'Stopped'
              : !(u.name in $codes) ? 'Loading...'
              : ($codes[u.name].length != 27) ? 'Booting'
              : $codes[u.name].length == 27 ? 'Running'
              : 'Loading...'
              }
            </div>
		    	</a>
	  		  <a href={u.name}>
  	    		<Fa icon={faGear} size="1.2x" />
  	    	</a>
			  </div>
			{/if}
		 {/each}

    {/if}

	</div>
<style>
	a { color: inherit; }

  .welcome {
    padding: 0 24px 0 24px;
    line-height: 24px;
    font-size: 14px;
  }

  .wrapper {
    margin-bottom: 28px;
    margin-top: 8px;
    display: flex;
    flex-direction: column;
    max-height: 264px;
    overflow: auto;
    -ms-overflow-style: none;
    scrollbar-width: none;
  }

  .wrapper::-webkit-scrollbar {
    display: none;
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
