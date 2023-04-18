
export const getCookie = (cookie, name) => {
  const cookieArray = cookie.split(";")
  for (let i = 0; i < cookieArray.length; i++) {
    let cookie = cookieArray[i]
    while (cookie.charAt(0) === " ") {
      cookie = cookie.substring(1)
    }
    if (cookie.indexOf(name) === 0) {
      return cookie.substring(name.length + 1, cookie.length)
    }
  }
  return ""
}

export const genRequestId = n => 
    [...Array(n)].map(() => Math.floor(Math.random() * 16).toString(16)).join('')
