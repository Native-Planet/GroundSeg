import nacl from 'tweetnacl';
import { encodeBase64 } from 'tweetnacl-util'

export const generateKeys = () => {
  const keys = nacl.box.keyPair()

  const pub = encodeBase64(keys.publicKey)
  const priv = encodeBase64(keys.secretKey)

  return {
    "pubkey":pub,
    "privkey":priv
  }
}
