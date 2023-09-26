//
//  Session
//

export const saveSession = async token => {
  if ((token.id === "") || (token.token === "")) {
    console.log("empty token field")
  } else {
    console.log("saving token")
    localStorage.setItem('id',token.id)
    localStorage.setItem('token',token.token)
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
    [...Array(n)].map(() => Math.floor(Math.random() * 16).toString(16)).join('')
