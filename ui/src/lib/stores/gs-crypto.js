//
//  Session
//

export const saveSession = async token => {
  if ((token.id === "") || (token.token === "")) {
    return false
  } else {
    localStorage.setItem('id',token.id)
    localStorage.setItem('token',token.token)
    return true
  }
}

export const loadSession = async () => {
  const id = localStorage.getItem('id')
  const token = localStorage.getItem('token')
  if ((id === null) || (token === null)) {
    return null
  }
  return {'id':id,'token':token}
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
    Array.from(crypto.getRandomValues(new Uint8Array(Math.ceil(n / 2))), byte =>
      byte.toString(16).padStart(2, '0')
    ).join('').slice(0, n)
