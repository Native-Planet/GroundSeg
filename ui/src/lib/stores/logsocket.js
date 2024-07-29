let SESSION;

// Initialize connection
export const connect = async url => {
  SESSION = new WebSocket(url);
  SESSION.onopen = () => handleOpen();
  SESSION.onmessage = (message) => handleMessage(JSON.parse(message.data));
  SESSION.onerror = (error) => console.log(error);
  SESSION.onclose = () => reconnect(url);
}

const handleOpen = () => {
  console.log("opened")
}

const handleMessage = msg => {
  console.log("msg: " + msg)
}

const reconnect = url => {
  console.log("disconnected")
  //connect(url)
}
