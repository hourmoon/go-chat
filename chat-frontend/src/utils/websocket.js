// WebSocket 管理工具
class WebSocketService {
  constructor() {
    this.socket = null
    this.reconnectAttempts = 0
    this.maxReconnectAttempts = 5
    this.reconnectInterval = 3000 // 3秒
  }

  connect(url) {
    return new Promise((resolve, reject) => {
      try {
        this.socket = new WebSocket(url)
        
        this.socket.onopen = () => {
          this.reconnectAttempts = 0
          resolve(this.socket)
        }
        
        this.socket.onerror = (error) => {
          reject(error)
        }
        
        this.socket.onclose = () => {
          // 自动重连逻辑
          if (this.reconnectAttempts < this.maxReconnectAttempts) {
            setTimeout(() => {
              this.reconnectAttempts++
              this.connect(url)
            }, this.reconnectInterval)
          }
        }
      } catch (error) {
        reject(error)
      }
    })
  }

  send(message) {
    if (this.socket && this.socket.readyState === WebSocket.OPEN) {
      this.socket.send(message)
      return true
    }
    return false
  }

  close() {
    if (this.socket) {
      this.socket.close()
      this.socket = null
    }
  }

  onMessage(callback) {
    if (this.socket) {
      this.socket.onmessage = callback
    }
  }
}

export default new WebSocketService()