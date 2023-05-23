export default class GroundSegJS {

  constructor(url) {
    this.connected = false;
    this.url = url;
    this.ws = new WebSocket(this.url);
    this.structure = {};
    this.activity = {}
  }

  connect() {
    return new Promise((resolve, reject) => {
      this.ws.onopen = () => {
        this.connected = true
        resolve(this.connected)
      };

      this.ws.onmessage = (event) => {
        this.updateData(event.data)
      };

      this.ws.onerror = (error) => {
        console.log("connection failed", error);
      };
    });
  };

  login(id,pwd="",token=null) {
    let data = {"id":id,"payload":{"category":"system","module":"login","action":{"password":pwd}}}
    this.send(data,token)
  }

  verify(id,token=null) {
    let data = {"id":id,"payload":{"category":"token","module":null,"action":null}}
    this.send(data,token)
  }

  send(data,token=null) {
    if (token) {
      data['token'] = token
    }
    console.log(data.id + " attempting to send message.." )
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
    if (data.hasOwnProperty('activity')) {
      this.activity = this.deepMerge(this.activity, data)
    } else {
      this.structure = this.deepMerge(this.structure, data);
    }
  };

  deepMerge(target, source) {
    for (const key in source) {
      if (typeof source[key] === 'object' && !Array.isArray(source[key]) && source[key] !== null) {
        if (!target.hasOwnProperty(key)) {
          target[key] = {};
        }
        this.deepMerge(target[key], source[key])
      } else {
        target[key] = source[key]
      }
    }
    return target
  }
}
