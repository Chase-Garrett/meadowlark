class MeadowlarkApp {
    constructor() {
        this.socket = null;
        this.currentRoom = null;
        this.currentRoomName = '';
        this.username = null;
        this.token = null;
        this.reconnectAttempts = 0;
        this.maxReconnectAttempts = 5;
        this.messageHistory = new Map();
    }

    init() {
        const savedToken = localStorage.getItem('meadowlark_token');
        const savedUsername = localStorage.getItem('meadowlark_username');
        
        if (savedToken && savedUsername) {
            this.token = savedToken;
            this.username = savedUsername;
            this.showApp();
        }
    }

    toggleAuthForm() {
        const loginForm = document.getElementById('loginForm');
        const registerForm = document.getElementById('registerForm');
        
        if (loginForm.style.display === 'none') {
            loginForm.style.display = 'block';
            registerForm.style.display = 'none';
        } else {
            loginForm.style.display = 'none';
            registerForm.style.display = 'block';
        }
        
        document.getElementById('loginAlert').classList.add('d-none');
        document.getElementById('registerAlert').classList.add('d-none');
    }

    async handleLogin() {
        const username = document.getElementById('loginUsername').value.trim();
        const password = document.getElementById('loginPassword').value;
        const alertDiv = document.getElementById('loginAlert');

        if (!username || !password) {
            this.showAlert(alertDiv, 'Please fill in all fields');
            return;
        }

        try {
            const response = await fetch('/api/login', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ username, password })
            });

            const data = await response.json();

            if (response.ok) {
                this.token = data.token;
                this.username = username;
                localStorage.setItem('meadowlark_token', this.token);
                localStorage.setItem('meadowlark_username', this.username);
                this.showApp();
            } else {
                this.showAlert(alertDiv, data.error || 'Login failed');
            }
        } catch (error) {
            this.showAlert(alertDiv, 'Connection error. Please try again.');
            console.error('Login error:', error);
        }
    }

    async handleRegister() {
        const username = document.getElementById('registerUsername').value.trim();
        const email = document.getElementById('registerEmail').value.trim();
        const password = document.getElementById('registerPassword').value;
        const alertDiv = document.getElementById('registerAlert');

        if (!username || !email || !password) {
            this.showAlert(alertDiv, 'Please fill in all fields');
            return;
        }

        if (password.length < 6) {
            this.showAlert(alertDiv, 'Password must be at least 6 characters');
            return;
        }

        try {
            const response = await fetch('/api/register', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ username, email, password })
            });

            const data = await response.json();

            if (response.ok) {
                this.showAlert(alertDiv, 'Registration successful! Please login.', 'success');
                setTimeout(() => this.toggleAuthForm(), 1500);
            } else {
                this.showAlert(alertDiv, data.error || 'Registration failed');
            }
        } catch (error) {
            this.showAlert(alertDiv, 'Connection error. Please try again.');
            console.error('Registration error:', error);
        }
    }

    showAlert(element, message, type = 'danger') {
        element.className = `alert alert-${type}`;
        element.textContent = message;
        element.classList.remove('d-none');
    }

    showApp() {
        document.getElementById('authContainer').style.display = 'none';
        document.getElementById('appContainer').style.display = 'block';
        document.getElementById('currentUsername').textContent = this.username;
        
        this.connectWebSocket();
        this.loadRooms();
    }

    logout() {
        localStorage.removeItem('meadowlark_token');
        localStorage.removeItem('meadowlark_username');
        if (this.socket) {
            this.socket.close();
        }
        location.reload();
    }

    connectWebSocket() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/ws`;

        this.socket = new WebSocket(wsUrl);

        this.socket.onopen = () => {
            console.log('WebSocket connected');
            this.updateConnectionStatus(true);
            this.reconnectAttempts = 0;
        };

        this.socket.onmessage = (event) => {
            const message = JSON.parse(event.data);
            this.displayMessage(message);
        };

        this.socket.onclose = () => {
            console.log('WebSocket disconnected');
            this.updateConnectionStatus(false);
            this.attemptReconnect();
        };

        this.socket.onerror = (error) => {
            console.error('WebSocket error:', error);
        };
    }

    attemptReconnect() {
        if (this.reconnectAttempts < this.maxReconnectAttempts) {
            this.reconnectAttempts++;
            console.log(`Reconnection attempt ${this.reconnectAttempts}/${this.maxReconnectAttempts}`);
            setTimeout(() => this.connectWebSocket(), 3000);
        }
    }

    updateConnectionStatus(connected) {
        const statusElement = document.getElementById('connectionStatus');
        if (connected) {
            statusElement.textContent = 'Connected';
            statusElement.className = 'connection-status status-connected';
        } else {
            statusElement.textContent = 'Disconnected';
            statusElement.className = 'connection-status status-disconnected';
        }
    }

    async loadRooms() {
        try {
            const response = await fetch('/api/rooms', {
                headers: { 'Authorization': `Bearer ${this.token}` }
            });
            const rooms = await response.json();
            this.displayRooms(rooms);
        } catch (error) {
            console.error('Error loading rooms:', error);
        }
    }

    displayRooms(rooms) {
        const roomList = document.getElementById('roomList');
        roomList.innerHTML = '';

        if (rooms.length === 0) {
            roomList.innerHTML = '<li class="text-muted p-3 text-center">No rooms yet. Create one!</li>';
            return;
        }

        rooms.forEach(room => {
            const li = document.createElement('li');
            li.className = 'room-item';
            if (room.id === this.currentRoom) {
                li.classList.add('active');
            }
            li.innerHTML = `
                <div class="room-name">${this.escapeHtml(room.name)}</div>
                ${room.description ? `<div class="room-description">${this.escapeHtml(room.description)}</div>` : ''}
            `;
            li.onclick = () => this.selectRoom(room.id, room.name, room.description || '');
            roomList.appendChild(li);
        });
    }

    async selectRoom(roomId, roomName, roomDescription) {
        this.currentRoom = roomId;
        this.currentRoomName = roomName;
        
        document.getElementById('currentRoomName').textContent = roomName;
        document.getElementById('currentRoomDescription').textContent = roomDescription;

        document.querySelectorAll('.room-item').forEach(item => {
            item.classList.remove('active');
        });
        event.currentTarget.classList.add('active');

        await this.loadMessages(roomId);
    }

    async loadMessages(roomId) {
        try {
            const response = await fetch(`/api/messages/${roomId}`, {
                headers: { 'Authorization': `Bearer ${this.token}` }
            });
            const messages = await response.json();
            
            const messageArea = document.getElementById('messageArea');
            messageArea.innerHTML = '';
            
            messages.forEach(msg => this.displayMessage(msg, false));
            this.scrollToBottom();
        } catch (error) {
            console.error('Error loading messages:', error);
        }
    }

    displayMessage(message, animate = true) {
        if (message.room_id !== this.currentRoom) return;

        const messageArea = document.getElementById('messageArea');
        const messageDiv = document.createElement('div');
        messageDiv.className = 'message';
        
        if (message.username === this.username) {
            messageDiv.classList.add('own-message');
        }

        const timestamp = new Date(message.timestamp).toLocaleTimeString([], { 
            hour: '2-digit', 
            minute: '2-digit' 
        });

        messageDiv.innerHTML = `
            <div class="message-header">
                <span class="message-username">${this.escapeHtml(message.username)}</span>
                <span class="message-timestamp">${timestamp}</span>
            </div>
            <div class="message-content">${this.escapeHtml(message.content)}</div>
        `;

        messageArea.appendChild(messageDiv);
        this.scrollToBottom();
    }

    sendMessage() {
        const input = document.getElementById('messageInput');
        const content = input.value.trim();

        if (!content || !this.currentRoom || !this.socket) return;

        const message = {
            username: this.username,
            room_id: this.currentRoom,
            content: content,
            timestamp: new Date().toISOString()
        };

        this.socket.send(JSON.stringify(message));
        input.value = '';
    }

    showCreateRoomModal() {
        const modal = new bootstrap.Modal(document.getElementById('createRoomModal'));
        modal.show();
    }

    async createRoom() {
        const name = document.getElementById('roomName').value.trim();
        const description = document.getElementById('roomDescription').value.trim();

        if (!name) {
            alert('Please enter a room name');
            return;
        }

        try {
            const response = await fetch('/api/rooms', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${this.token}`
                },
                body: JSON.stringify({ name, description })
            });

            if (response.ok) {
                const modal = bootstrap.Modal.getInstance(document.getElementById('createRoomModal'));
                modal.hide();
                document.getElementById('roomName').value = '';
                document.getElementById('roomDescription').value = '';
                await this.loadRooms();
            }
        } catch (error) {
            console.error('Error creating room:', error);
            alert('Failed to create room. Please try again.');
        }
    }

    scrollToBottom() {
        const messageArea = document.getElementById('messageArea');
        messageArea.scrollTop = messageArea.scrollHeight;
    }

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
}

// Initialize the app when the DOM is loaded
const app = new MeadowlarkApp();
app.init();