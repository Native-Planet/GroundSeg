<script>
  import { url } from '/src/Scripts/server'
  export let name, key

  const boot = () => {
    const f = new FormData()
    f.append("patp", name)
    f.append("key", key)
    const r = fetch(url, {
      method: 'POST',
      body: f
    })
    // handle redirect after success
    // .then(res => console.log(res))
  }
</script>

<div>
  <a href="/">Cancel</a>
  <button on:click={boot}>
    Boot Ship
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
