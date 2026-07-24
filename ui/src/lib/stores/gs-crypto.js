//
//  Session
//

const SESSION_ID_KEY = 'id'
const SESSION_TOKEN_KEY = 'token'
const REMEMBER_SESSION_KEY = 'rememberSession'

const hasToken = storage => (
  storage.getItem(SESSION_ID_KEY) !== null &&
  storage.getItem(SESSION_TOKEN_KEY) !== null
)

const clearToken = storage => {
  storage.removeItem(SESSION_ID_KEY)
  storage.removeItem(SESSION_TOKEN_KEY)
}

export const isSessionRemembered = () => (
  localStorage.getItem(REMEMBER_SESSION_KEY) === 'true' ||
  hasToken(localStorage)
)

export const setSessionRemembered = remember => {
  if (remember) {
    localStorage.setItem(REMEMBER_SESSION_KEY, 'true')
  } else {
    const id = localStorage.getItem(SESSION_ID_KEY)
    const token = localStorage.getItem(SESSION_TOKEN_KEY)
    if ((id !== null) && (token !== null) && !hasToken(sessionStorage)) {
      sessionStorage.setItem(SESSION_ID_KEY, id)
      sessionStorage.setItem(SESSION_TOKEN_KEY, token)
    }
    clearToken(localStorage)
    localStorage.removeItem(REMEMBER_SESSION_KEY)
  }
}

export const saveSession = async (token, remember = isSessionRemembered()) => {
  if ((token.id === "") || (token.token === "")) {
    console.log("empty token field")
  } else {
    console.log("saving token")
    const targetStorage = remember ? localStorage : sessionStorage
    const staleStorage = remember ? sessionStorage : localStorage
    targetStorage.setItem(SESSION_ID_KEY, token.id)
    targetStorage.setItem(SESSION_TOKEN_KEY, token.token)
    clearToken(staleStorage)
    setSessionRemembered(remember)
  }
}

export const loadSession = async () => {
  const sessionID = sessionStorage.getItem(SESSION_ID_KEY)
  const sessionToken = sessionStorage.getItem(SESSION_TOKEN_KEY)
  if ((sessionID !== null) && (sessionToken !== null)) {
    return {'id':sessionID,'token':sessionToken}
  }

  const id = localStorage.getItem(SESSION_ID_KEY)
  const token = localStorage.getItem(SESSION_TOKEN_KEY)
  if ((id === null) || (token === null)) {
    return null
  }
  return {'id':id,'token':token}
}

export const clearSession = async () => {
  clearToken(sessionStorage)
  clearToken(localStorage)
  localStorage.removeItem(REMEMBER_SESSION_KEY)
}

export const toBase64 = arrayBuffer => {
  let binaryString = '';
  for (let i = 0; i < arrayBuffer.length; i++) {
        binaryString += String.fromCharCode(arrayBuffer[i]);
      }
  return btoa(binaryString);
}

//
//  Misc
//

export const generateRandom = n => 
    [...Array(n)].map(() => Math.floor(Math.random() * 16).toString(16)).join('')
