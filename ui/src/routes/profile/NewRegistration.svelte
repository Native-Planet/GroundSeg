<script>
  let regions = ['us-east','us-west','asia','europe']
  let region = 'us-west'
  let addr = "api.startram.io"
</script>
    <div class="body">

      <div class="region-wrapper">
        <div class="header">Select Region</div>
        <div class="regions">
          {#each regions as r}
            <div class="region" class:highlight={r == region}>{r}</div>
          {/each}
        </div>
      </div>

      <div class="spacer"></div>

      <div class="reg-wrapper">
        <div class="header">Register New StarTram Key</div>
        <div class="reg">
          <input class="reg-input" placeholder="NativePlanet-some-word-another-word" type="password" />
          <button class="reg-btn">Register</button>
        </div>
        <div class="header">Modify Endpoint (optional)</div>
        <div class="reg">
          <input class="reg-input" bind:value={addr} type="text" />
          <button class="reg-btn">Set</button>
        </div>
      </div>
    </div>
