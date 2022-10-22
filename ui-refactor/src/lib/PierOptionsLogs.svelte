<script>
  /*
  const exportLog = c => {
    const u = $api + "/settings/logs"
    const f = new FormData()
    f.append('logs', patp)
    fetch(u, {method: 'POST', body: f})
      .then(r => r.json()).then(d => {
          var element = document.createElement('a')
          element.setAttribute('href', 'data:text/plain;charset=utf-8,' + encodeURIComponent(d))
          element.setAttribute('download', 'logs_' + patp)
          element.style.display = 'none'
          document.body.appendChild(element)
          element.click()
          document.body.removeChild(element)
    })}
    */
</script>
todo
<!--
<div class="info">
	<div class="title">Logs</div>
  <div class="button-wrapper">
		<PrimaryButton
			standard="View Urbit Pier Logs"
			noMargin={true}
			on:click={()=>dispatch('toggleLogs')} />
		<PrimaryButton
			standard="Export Urbit Pier Logs" 
	 		noMargin={true}
			background='#FFFFFF4D'
		 	on:click={exportLog} />
  </div>
</div>
-->
