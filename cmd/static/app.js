class MeadowlarkApp {
    constructor() {
        this.socket = null;
        this.currentRecipient = null;
        this.currentRecipientName = '';
        this.username = null;
        this.token = null;
        this.reconnectAttempts = 0;
        this.maxReconnectAttempts = 5;
        this.messageHistory = new Map(); // Map<recipient, messages[]>
        this.users = [];
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
                this.username = data.username;
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
        this.loadUsers();
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
        const wsUrl = `${protocol}//${window.location.host}/ws?token=${encodeURIComponent(this.token)}`;

        this.socket = new WebSocket(wsUrl);

        this.socket.onopen = () => {
            console.log('WebSocket connected');
            this.updateConnectionStatus(true);
            this.reconnectAttempts = 0;
        };

        this.socket.onmessage = (event) => {
            try {
                const message = JSON.parse(event.data);
                this.handleIncomingMessage(message);
            } catch (error) {
                console.error('Error parsing WebSocket message:', error);
            }
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

    handleIncomingMessage(message) {
        // Backend sends: { Recipient, Sender, Content: []byte }
        // We need to handle the Content as base64 or UTF-8
        
        const sender = message.sender || message.Sender;
        const recipient = message.recipient || message.Recipient;
        
        // Determine if this message is for the current conversation
        const isForCurrentConversation = 
            (sender === this.currentRecipient && recipient === this.username) ||
            (sender === this.username && recipient === this.currentRecipient);

        // Decode content (assuming it's base64 encoded bytes)
        let content = '';
        if (message.content || message.Content) {
            const contentBytes = message.content || message.Content;
            if (typeof contentBytes === 'string') {
                // If it's a base64 string, decode it
                try {
                    const decoded = atob(contentBytes);
                    content = decoded;
                } catch {
                    // If not base64, treat as plain string
                    content = contentBytes;
                }
            } else if (Array.isArray(contentBytes)) {
                // If it's an array of bytes
                content = String.fromCharCode(...contentBytes);
            }
        }

        // Add to message history
        const conversationKey = sender === this.username ? recipient : sender;
        if (!this.messageHistory.has(conversationKey)) {
            this.messageHistory.set(conversationKey, []);
        }
        
        const messageObj = {
            sender: sender,
            recipient: recipient,
            content: content,
            timestamp: new Date().toISOString()
        };
        
        this.messageHistory.get(conversationKey).push(messageObj);

        // Display if it's for the current conversation
        if (isForCurrentConversation) {
            this.displayMessage(messageObj);
        } else {
            // Show notification or update UI for unread messages
            this.updateUserUnreadStatus(sender);
        }
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

    async loadUsers() {
        try {
            const response = await fetch('/api/users', {
                headers: { 'Authorization': `Bearer ${this.token}` }
            });
            
            if (!response.ok) {
                throw new Error('Failed to load users');
            }
            
            const users = await response.json();
            // Filter out current user
            this.users = users.filter(u => u !== this.username);
            this.displayUsers(this.users);
        } catch (error) {
            console.error('Error loading users:', error);
        }
    }

    displayUsers(users) {
        const userList = document.getElementById('userList');
        userList.innerHTML = '';

        if (users.length === 0) {
            userList.innerHTML = '<li class="text-muted p-3 text-center">No other users yet.</li>';
            return;
        }

        users.forEach(username => {
            const li = document.createElement('li');
            li.className = 'room-item'; // Reuse room-item class
            if (username === this.currentRecipient) {
                li.classList.add('active');
            }
            li.innerHTML = `
                <div class="room-name">${this.escapeHtml(username)}</div>
            `;
            li.onclick = () => this.selectUser(username);
            userList.appendChild(li);
        });
    }

    selectUser(username) {
        this.currentRecipient = username;
        this.currentRecipientName = username;
        
        document.getElementById('currentRoomName').textContent = `Chat with ${username}`;
        document.getElementById('currentRoomDescription').textContent = '';

        // Update active state
        document.querySelectorAll('.room-item').forEach(item => {
            item.classList.remove('active');
        });
        
        // Find and activate the clicked item
        const items = document.querySelectorAll('.room-item');
        items.forEach(item => {
            if (item.textContent.trim() === username) {
                item.classList.add('active');
            }
        });

        this.displayConversation(username);
    }

    displayConversation(username) {
        const messageArea = document.getElementById('messageArea');
        messageArea.innerHTML = '';
        
        const messages = this.messageHistory.get(username) || [];
        
        messages.forEach(msg => this.displayMessage(msg, false));
        this.scrollToBottom();
    }

    displayMessage(message, animate = true) {
        const messageArea = document.getElementById('messageArea');
        const messageDiv = document.createElement('div');
        messageDiv.className = 'message';
        
        if (message.sender === this.username) {
            messageDiv.classList.add('own-message');
        }

        const timestamp = new Date(message.timestamp).toLocaleTimeString([], { 
            hour: '2-digit', 
            minute: '2-digit' 
        });

        messageDiv.innerHTML = `
            <div class="message-header">
                <span class="message-username">${this.escapeHtml(message.sender)}</span>
                <span class="message-timestamp">${timestamp}</span>
            </div>
            <div class="message-content">${this.escapeHtml(message.content)}</div>
        `;

        messageArea.appendChild(messageDiv);
        if (animate) {
            this.scrollToBottom();
        }
    }

    sendMessage() {
        const input = document.getElementById('messageInput');
        const content = input.value.trim();

        if (!content || !this.currentRecipient || !this.socket) return;

        // Backend expects: { Recipient, Sender, Content: []byte }
        // Send content as plain string, backend will convert to []byte
        const message = {
            recipient: this.currentRecipient,
            sender: this.username,
            content: content
        };

        // Also add to local history immediately for better UX
        const conversationKey = this.currentRecipient;
        if (!this.messageHistory.has(conversationKey)) {
            this.messageHistory.set(conversationKey, []);
        }
        
        const messageObj = {
            sender: this.username,
            recipient: this.currentRecipient,
            content: content,
            timestamp: new Date().toISOString()
        };
        
        this.messageHistory.get(conversationKey).push(messageObj);
        this.displayMessage(messageObj);

        this.socket.send(JSON.stringify(message));
        input.value = '';
    }

    updateUserUnreadStatus(username) {
        // Could add visual indicator for unread messages
        // For now, just a placeholder
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
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', () => app.init());
} else {
    app.init();
}
