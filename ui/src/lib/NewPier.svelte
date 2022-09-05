<script>
  import { api } from '$lib/api'
  export let name, key, loading = false

  const boot = () => {
    loading = true
    const f = new FormData()
    const u = api + "/upload/key"
    f.append("patp", name)
    f.append("key", key)
    fetch(u, {method: 'POST',body: f})
      .then(d => d.json())
      .then(res => {
        if (res === 200) {
          window.location.href = "/" + name
        }
      })
  }
</script>

<div>
  <a href="/">Cancel</a>
  <button on:click={boot}>
    {loading ? "Creating.." : "Create Pier"}
  </button>
</div>

<style>
  div {
    display: flex;
    margin-top: 8px;
  }

  a {
    padding: 12px 20px 12px 20px;
    color: inherit;
    text-align: center;
    font-weight: 500;
    border-radius: 8px;
    background: #FFFFFF4D; 
    font-size: 16px;
  }

  button {
    padding: 12px 20px 12px 20px;
    font-weight: 500;
    background: #28B8ED;
    color: inherit;
    border-radius: 8px;
    border: none;
    font-size: 16px;
    font-family: inherit;
    margin-left: auto;
    cursor: pointer;

  }

</style>
