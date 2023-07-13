export default class GroundSegJS {
  constructor(url) {
    this.connected = false;
    this.url = url;
    this.structure = {};
    this.activity = {}
  }

  // Connect to websocket API
  connect() {
    console.log("attempting to connect..")
    this.ws = new WebSocket(this.url);
    return new Promise((resolve, reject) => {
      this.ws.onopen = () => {
        this.connected = true
        console.log("connected")
        resolve(this.connected)
      };
      this.ws.onmessage = (event) => {
        this.updateData(event.data)
      };
      this.ws.onerror = (error) => {
        console.log("connection failed", error);
      };
      this.ws.onclose = () => {
        this.connected = false
        console.log("closed")
      };
    });
  };

  // Login macro
  login(id,pwd="",token=null) {
    let data = {"id":id,"payload":{"category":"system","module":"login","action":{"password":pwd}}}
    console.log(id + " attempting to login.." )
    this.silentSend(data,token)
  }

  // Send token for verification
  verify(id,token=null) {
    let data = {"id":id,"payload":{"category":"token","module":null,"action":null}}
    console.log(id + " attempting to verify token.." )
    this.silentSend(data,token)
  }

  // Update form for session
  updateForm(id,template,item,value,token=null) {
    let data = {"id":id,"payload":{"category":"forms","template":template,"item":item,"value":value}}
    console.log(id + " updating " + template + " form" )
    this.silentSend(data,token)
  }

  //
  //  Setup
  //

  beginSetup(id,token=null) {
    let data = {"id":id,"payload":{"category":"system","module":"setup","action":"start"}}
    console.log(id + " starting setup" )
    this.silentSend(data,token)
  }

  //
  //  StarTram
  //

  // Update StarTram endpoint in config
  starTramEndpoint(id,token=null) {
    let data = {"id":id,"payload":{"category":"system","module":"startram","action":"endpoint"}}
    console.log(id + " attempting to update StarTram Endpoint")
    this.silentSend(data,token)
  }

  // Register startram
  starTramRegister(id,token=null) {
    let data = {"id":id,"payload":{"category":"system","module":"startram","action":"register"}}
    console.log(id + " attempting to register StarTram")
    this.silentSend(data,token)
  }

  // Stop startram
  starTramStop(id,token=null) {
    let data = {"id":id,"payload":{"category":"system","module":"startram","action":"stop"}}
    console.log(id + " attempting to stop StarTram")
    this.silentSend(data,token)
  }

  // Start startram
  starTramStart(id,token=null) {
    let data = {"id":id,"payload":{"category":"system","module":"startram","action":"start"}}
    console.log(id + " attempting to start StarTram")
    this.silentSend(data,token)
  }

  // Restart startram
  starTramRestart(id,token=null) {
    let data = {"id":id,"payload":{"category":"system","module":"startram","action":"restart"}}
    console.log(id + " attempting to restart StarTram")
    this.silentSend(data,token)
  }

  // Cancel startram subscription
  starTramCancel(id,token=null) {
    let data = {"id":id,"payload":{"category":"system","module":"startram","action":"cancel"}}
    console.log(id + " attempting to cancel StarTram Subscription")
    this.silentSend(data,token)
  }

  //
  //  Urbit
  //

  // Toggle Urbit network
  urbitsAccessToggle(id,ship,token=null) {
    let data = {"id":id,"payload":{"category":"urbits","ship":ship,"module":"access","action":"toggle"}}
    console.log(id + ":" + ship + " attempting to toggle network")
    this.silentSend(data,token)
  }

  // Meld from Urth
  urbitsMeldUrth(id,ship,token=null) {
    let data = {"id":id,"payload":{"category":"urbits","ship":ship,"module":"meld","action":"urth"}}
    console.log(id + ":" + ship + " attempting to meld from urth ")
    this.silentSend(data,token)
  }

  // Send raw action
  send(data,token=null) {
    if (token) {
      data['token'] = token
    }
    console.log(data.id + " attempting to send message.." )
    this.ws.send(JSON.stringify(data));
  }

  // Same as send but without logging
  silentSend(data,token=null) {
    if (token) {
      data['token'] = token
    }
    this.ws.send(JSON.stringify(data));
  }

  close() {
    this.ws.close();
  }

  deleteActivity(id) {
    if (id) {
      delete this.activity.activity[id]
      console.log(id + " activity acknowledged" )
    }
  }

  updateData(data) {
    data = JSON.parse(data)
    if (data.type == "activity") {
      this.activity = this.deepMerge(this.activity, data)
    } else {
      this.structure = this.deepMerge(this.structure, data);
    }
  };

  deepMerge(target, source) {
    for (const key in source) {
      if (typeof source[key] === 'object' && !Array.isArray(source[key]) && source[key] !== null) {
        if (target !== null) {
          if ((target !== null) && (!target.hasOwnProperty(key))) {
            target[key] = {};
          }
          this.deepMerge(target[key], source[key])
        }
      } else {
        target[key] = source[key]
      }
    }
    return target
  }
}
