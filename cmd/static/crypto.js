// Cryptography utilities for end-to-end encryption using Web Crypto API

class CryptoUtils {
    constructor() {
        this.algorithm = {
            name: 'RSA-OAEP',
            modulusLength: 2048,
            publicExponent: new Uint8Array([1, 0, 1]), // 65537
            hash: 'SHA-256'
        };
        this.privateKeyStorageKey = 'meadowlark_private_key';
        this.publicKeyStorageKey = 'meadowlark_public_key';
    }

    /**
     * Generate a new RSA key pair for encryption
     * @returns {Promise<CryptoKeyPair>}
     */
    async generateKeyPair() {
        try {
            const keyPair = await crypto.subtle.generateKey(
                this.algorithm,
                true, // extractable
                ['encrypt', 'decrypt']
            );
            return keyPair;
        } catch (error) {
            console.error('Error generating key pair:', error);
            throw new Error('Failed to generate encryption keys');
        }
    }

    /**
     * Export public key as base64 string
     * @param {CryptoKey} publicKey 
     * @returns {Promise<string>}
     */
    async exportPublicKey(publicKey) {
        try {
            const exported = await crypto.subtle.exportKey('spki', publicKey);
            return this.arrayBufferToBase64(exported);
        } catch (error) {
            console.error('Error exporting public key:', error);
            throw new Error('Failed to export public key');
        }
    }

    /**
     * Export private key as base64 string
     * @param {CryptoKey} privateKey 
     * @returns {Promise<string>}
     */
    async exportPrivateKey(privateKey) {
        try {
            const exported = await crypto.subtle.exportKey('pkcs8', privateKey);
            return this.arrayBufferToBase64(exported);
        } catch (error) {
            console.error('Error exporting private key:', error);
            throw new Error('Failed to export private key');
        }
    }

    /**
     * Import public key from base64 string
     * @param {string} base64Key 
     * @returns {Promise<CryptoKey>}
     */
    async importPublicKey(base64Key) {
        try {
            const keyData = this.base64ToArrayBuffer(base64Key);
            const publicKey = await crypto.subtle.importKey(
                'spki',
                keyData,
                this.algorithm,
                false, // not extractable for security
                ['encrypt']
            );
            return publicKey;
        } catch (error) {
            console.error('Error importing public key:', error);
            throw new Error('Failed to import public key');
        }
    }

    /**
     * Import private key from base64 string
     * @param {string} base64Key 
     * @returns {Promise<CryptoKey>}
     */
    async importPrivateKey(base64Key) {
        try {
            const keyData = this.base64ToArrayBuffer(base64Key);
            const privateKey = await crypto.subtle.importKey(
                'pkcs8',
                keyData,
                this.algorithm,
                false, // not extractable for security
                ['decrypt']
            );
            return privateKey;
        } catch (error) {
            console.error('Error importing private key:', error);
            throw new Error('Failed to import private key');
        }
    }

    /**
     * Encrypt a message using recipient's public key
     * @param {string} plaintext 
     * @param {CryptoKey} publicKey 
     * @returns {Promise<ArrayBuffer>}
     */
    async encrypt(plaintext, publicKey) {
        try {
            const encoder = new TextEncoder();
            const data = encoder.encode(plaintext);
            
            // RSA-OAEP can encrypt up to 190 bytes (with SHA-256)
            // For longer messages, we need to split or use hybrid encryption
            // For now, we'll limit message size or use AES-GCM hybrid encryption
            
            if (data.length > 190) {
                // Use hybrid encryption: AES-GCM for content, RSA for AES key
                return await this.encryptHybrid(plaintext, publicKey);
            }
            
            const encrypted = await crypto.subtle.encrypt(
                {
                    name: 'RSA-OAEP'
                },
                publicKey,
                data
            );
            
            return encrypted;
        } catch (error) {
            console.error('Error encrypting message:', error);
            throw new Error('Failed to encrypt message');
        }
    }

    /**
     * Hybrid encryption for messages longer than RSA-OAEP limit
     * Uses AES-GCM for content, RSA-OAEP for AES key
     */
    async encryptHybrid(plaintext, publicKey) {
        // Generate AES key
        const aesKey = await crypto.subtle.generateKey(
            { name: 'AES-GCM', length: 256 },
            true,
            ['encrypt']
        );
        
        // Export AES key
        const exportedAesKey = await crypto.subtle.exportKey('raw', aesKey);
        
        // Encrypt AES key with RSA
        const encryptedAesKey = await crypto.subtle.encrypt(
            { name: 'RSA-OAEP' },
            publicKey,
            exportedAesKey
        );
        
        // Generate IV for AES-GCM
        const iv = crypto.getRandomValues(new Uint8Array(12));
        
        // Encrypt message with AES-GCM
        const encoder = new TextEncoder();
        const data = encoder.encode(plaintext);
        const encrypted = await crypto.subtle.encrypt(
            { name: 'AES-GCM', iv: iv },
            aesKey,
            data
        );
        
        // Combine: encrypted AES key + IV + encrypted data
        const result = new Uint8Array(encryptedAesKey.length + iv.length + encrypted.byteLength);
        result.set(new Uint8Array(encryptedAesKey), 0);
        result.set(iv, encryptedAesKey.length);
        result.set(new Uint8Array(encrypted), encryptedAesKey.length + iv.length);
        
        return result.buffer;
    }

    /**
     * Decrypt a message using own private key
     * @param {ArrayBuffer} encryptedData 
     * @param {CryptoKey} privateKey 
     * @returns {Promise<string>}
     */
    async decrypt(encryptedData, privateKey) {
        try {
            const view = new Uint8Array(encryptedData);
            
            // Check if it's hybrid encryption (starts with RSA-encrypted AES key)
            // RSA-OAEP encrypted 32-byte AES key is 256 bytes
            if (view.length > 256) {
                // Try hybrid decryption
                try {
                    return await this.decryptHybrid(encryptedData, privateKey);
                } catch (e) {
                    // Fall through to try direct RSA decryption
                }
            }
            
            // Direct RSA-OAEP decryption
            const decrypted = await crypto.subtle.decrypt(
                {
                    name: 'RSA-OAEP'
                },
                privateKey,
                encryptedData
            );
            
            const decoder = new TextDecoder();
            return decoder.decode(decrypted);
        } catch (error) {
            console.error('Error decrypting message:', error);
            throw new Error('Failed to decrypt message. This might be encrypted with a different key.');
        }
    }

    /**
     * Hybrid decryption for messages longer than RSA-OAEP limit
     */
    async decryptHybrid(encryptedData, privateKey) {
        const view = new Uint8Array(encryptedData);
        
        // Extract encrypted AES key (first 256 bytes)
        const encryptedAesKey = view.slice(0, 256).buffer;
        
        // Decrypt AES key with RSA
        const decryptedAesKey = await crypto.subtle.decrypt(
            { name: 'RSA-OAEP' },
            privateKey,
            encryptedAesKey
        );
        
        // Import AES key
        const aesKey = await crypto.subtle.importKey(
            'raw',
            decryptedAesKey,
            { name: 'AES-GCM' },
            false,
            ['decrypt']
        );
        
        // Extract IV (next 12 bytes)
        const iv = view.slice(256, 256 + 12);
        
        // Extract encrypted content (rest)
        const encrypted = view.slice(256 + 12).buffer;
        
        // Decrypt with AES-GCM
        const decrypted = await crypto.subtle.decrypt(
            { name: 'AES-GCM', iv: iv },
            aesKey,
            encrypted
        );
        
        const decoder = new TextDecoder();
        return decoder.decode(decrypted);
    }

    /**
     * Store keys in localStorage (private key) and IndexedDB (more secure)
     */
    async storeKeys(keyPair) {
        try {
            const publicKeyBase64 = await this.exportPublicKey(keyPair.publicKey);
            const privateKeyBase64 = await this.exportPrivateKey(keyPair.privateKey);
            
            // Store public key in localStorage (can be shared)
            localStorage.setItem(this.publicKeyStorageKey, publicKeyBase64);
            
            // Store private key in localStorage (in production, use IndexedDB or encrypted storage)
            localStorage.setItem(this.privateKeyStorageKey, privateKeyBase64);
            
            return {
                publicKey: publicKeyBase64,
                privateKey: privateKeyBase64
            };
        } catch (error) {
            console.error('Error storing keys:', error);
            throw new Error('Failed to store encryption keys');
        }
    }

    /**
     * Load keys from storage
     * @returns {Promise<{publicKey: CryptoKey, privateKey: CryptoKey}>}
     */
    async loadKeys() {
        try {
            const publicKeyBase64 = localStorage.getItem(this.publicKeyStorageKey);
            const privateKeyBase64 = localStorage.getItem(this.privateKeyStorageKey);
            
            if (!publicKeyBase64 || !privateKeyBase64) {
                return null;
            }
            
            const publicKey = await this.importPublicKey(publicKeyBase64);
            const privateKey = await this.importPrivateKey(privateKeyBase64);
            
            return { publicKey, privateKey };
        } catch (error) {
            console.error('Error loading keys:', error);
            return null;
        }
    }

    /**
     * Get stored public key as base64 string
     * @returns {string|null}
     */
    getStoredPublicKeyBase64() {
        return localStorage.getItem(this.publicKeyStorageKey);
    }

    /**
     * Clear all stored keys
     */
    clearKeys() {
        localStorage.removeItem(this.publicKeyStorageKey);
        localStorage.removeItem(this.privateKeyStorageKey);
    }

    // Utility functions
    arrayBufferToBase64(buffer) {
        const bytes = new Uint8Array(buffer);
        let binary = '';
        for (let i = 0; i < bytes.byteLength; i++) {
            binary += String.fromCharCode(bytes[i]);
        }
        return btoa(binary);
    }

    base64ToArrayBuffer(base64) {
        const binary = atob(base64);
        const bytes = new Uint8Array(binary.length);
        for (let i = 0; i < binary.length; i++) {
            bytes[i] = binary.charCodeAt(i);
        }
        return bytes.buffer;
    }
}

// Export for use in app.js
const cryptoUtils = new CryptoUtils();

