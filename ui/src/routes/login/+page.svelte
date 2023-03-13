<script>
  import { scale } from 'svelte/transition'
  import { onMount, onDestroy } from 'svelte'
  import { api, updateState } from '$lib/api'
  import { page } from '$app/stores'

	import Card from '$lib/Card.svelte'
  import PrimaryButton from '$lib/PrimaryButton.svelte'
  import EyeButton from '$lib/EyeButton.svelte'

  export let data
  updateState(data)

  let inView = false
  let showLogin = false
  let pwdView = false
  let loginPassword = ''
  let buttonStatus = 'standard'
  let pubKey = ''

	onDestroy(()=> inView = false)
  onMount(()=> {
    if (data['status'] == 200) {
      console.log("logged in")
      window.location.href = "/"
    } else if (data['status'] == 'setup') {
      window.location.href = "/setup"
    } else {
      console.log(data['status'])
    }
    inView = true
    getLoginKey()
  })

  const getLoginKey = () => {
    if (inView && ($page.url.pathname == "/login")) {
      fetch($api + '/login/key')
      .then(r => r.json())
        .then(d => {
          pubKey = d
        })
      setTimeout(getLoginKey, 30000)
  }}

  const togglePwdView = () => {
    pwdView = !pwdView
    document.querySelector('#login-password').type = pwdView ? 'text' : 'password'
  }

  const handleLogin = async () => {
    buttonStatus = 'loading'
    const enc_pwd = await encryptPassword()

    fetch($api + '/login', {
			method: 'POST',
      headers: {"Content-Type" : "application/json"},
      credentials : "include",
			body: JSON.stringify({'password':enc_pwd})
	  })
      .then(r => r.json())
      .then(d => { 
        if (d == 200) {
          buttonStatus = 'success'
          setTimeout(()=> window.location.href = '/', 1000)
        } else {
          console.log(d)
          buttonStatus = 'failure'
          setTimeout(()=> {
            buttonStatus = 'standard'
            loginPassword = ''
          }, 2000)
        }
    })
  }

  const encryptPassword = async () => {
    // encode password
    const password = new TextEncoder().encode(loginPassword.trim())

    // encode pubkey
    const binaryString = atob(pubKey)
    const bytes = new Uint8Array(binaryString.length)
    for (let i = 0; i < binaryString.length; i++) {
      bytes[i] = binaryString.charCodeAt(i)
    }
    const publicKey = await crypto.subtle.importKey(
      "spki", bytes, { name: "RSA-OAEP", hash: "SHA-256"}, true,["encrypt"]
    )

    // encrypt password
    const ciphertext = await crypto.subtle.encrypt(
      { name: "RSA-OAEP" },
      publicKey,
      password
    )

    // encode password to b64
    return await btoa(String.fromCharCode(...new Uint8Array(ciphertext)))
  }


</script>

{#if inView}
  <Card width="640px">
    <div class="main-wrapper">
      <div class="opened-wrapper" in:scale={{delay:400, duration: 120}}>
        <img src="/npfull.svg" alt="Native Planet Logo" />
        <div class="login-wrapper">
          <input
            id="login-password"
            bind:value={loginPassword}
            class="login-password"
            type="password"
            placeholder='Password'
            disabled={buttonStatus != 'standard'}
          />
        </div>
        <PrimaryButton
          top=24
          left={false}
          standard="Login"
          success="Login successful!"
          failure="Login failed"
          loading="Logging you in.."
          status={(loginPassword.length > 0) && (pubKey.length > 0) ? buttonStatus : 'disabled'}
          on:click={handleLogin}
        />
      </div>
    </div>
  </Card>
{/if}

<style>
  .main-wrapper {
    text-align: center;
    margin: 40px 80px;
  }
  .login-wrapper {
    margin-top: 24px;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }
  .login-password {
    font-size: 12px;
    padding: 8px;
    background: #ffffff4d;
    border-radius: 6px;
    flex: 1;
    border: none;
    font-family: inherit;
    color: inherit;
  }
  input {
    text-align: center;
  }
  input:focus {
    outline: none;
  }
  input:disabled {
    opacity: 0.4;
  }
  input::placeholder {
    color: inherit;
    opacity: .6;
  }
</style>
