//
//  Session
//

export const saveSession = token => {
  localStorage.setItem('id',token.id)
  localStorage.setItem('token',token.token)
}

export const loadSession = () => {
  const id = localStorage.getItem('id')
  const token = localStorage.getItem('token')
  if ((id === null) || (token === null)) {
    return null
  }
  return {'id':id,'token':token}
}

//
//  Misc
//

export const generateRandom = n => 
    [...Array(n)].map(() => Math.floor(Math.random() * 16).toString(16)).join('')
