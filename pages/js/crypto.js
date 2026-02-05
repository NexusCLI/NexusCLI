/**
 * Shared cryptography utilities for Zephyrus pages
 * Provides AES-GCM encryption/decryption, PBKDF2 key derivation
 */

const CRYPTO = {
    SALT_SIZE: 16,
    NONCE_SIZE: 12,
    ITERATIONS: 100000,
    KEY_SIZE: 256,

    /**
     * Decrypt a file using a password (with PBKDF2 key derivation)
     * Format: [Salt (16 bytes)][Nonce (12 bytes)][Ciphertext]
     */
    async decryptWithPassword(encryptedData, password) {
        const view = new Uint8Array(encryptedData);
        const salt = view.slice(0, this.SALT_SIZE);
        const nonce = view.slice(this.SALT_SIZE, this.SALT_SIZE + this.NONCE_SIZE);
        const ciphertext = view.slice(this.SALT_SIZE + this.NONCE_SIZE);

        // Derive key using PBKDF2
        const keyMaterial = await window.crypto.subtle.importKey(
            'raw',
            new TextEncoder().encode(password),
            'PBKDF2',
            false,
            ['deriveBits']
        );

        const derivedBits = await window.crypto.subtle.deriveBits(
            {
                name: 'PBKDF2',
                salt: salt,
                iterations: this.ITERATIONS,
                hash: 'SHA-256'
            },
            keyMaterial,
            this.KEY_SIZE
        );

        const key = await window.crypto.subtle.importKey(
            'raw',
            derivedBits,
            'AES-GCM',
            false,
            ['decrypt']
        );

        // Decrypt using AES-GCM
        const decrypted = await window.crypto.subtle.decrypt(
            {
                name: 'AES-GCM',
                iv: nonce
            },
            key,
            ciphertext
        );

        return new Uint8Array(decrypted);
    },

    /**
     * Decrypt a file using a raw 32-byte key (no PBKDF2)
     * Format: [Nonce (12 bytes)][Ciphertext]
     */
    async decryptWithKey(encryptedData, keyBuffer) {
        const view = new Uint8Array(encryptedData);
        const nonce = view.slice(0, this.NONCE_SIZE);
        const ciphertext = view.slice(this.NONCE_SIZE);

        // Import the key directly
        const key = await window.crypto.subtle.importKey(
            'raw',
            keyBuffer,
            'AES-GCM',
            false,
            ['decrypt']
        );

        // Decrypt using AES-GCM
        const decrypted = await window.crypto.subtle.decrypt(
            {
                name: 'AES-GCM',
                iv: nonce
            },
            key,
            ciphertext
        );

        return new Uint8Array(decrypted);
    },

    /**
     * Convert a hex string to Uint8Array
     */
    hexToBuffer(hexString) {
        const bytes = new Uint8Array(hexString.length / 2);
        for (let i = 0; i < hexString.length; i += 2) {
            bytes[i / 2] = parseInt(hexString.substr(i, 2), 16);
        }
        return bytes;
    },

    /**
     * Convert Uint8Array to hex string
     */
    bufferToHex(buffer) {
        return Array.from(new Uint8Array(buffer))
            .map(b => b.toString(16).padStart(2, '0'))
            .join('');
    },

    /**
     * Determine MIME type from file extension
     */
    getMimeType(filename) {
        const ext = filename.toLowerCase().split('.').pop();
        const mimeTypes = {
            'txt': 'text/plain',
            'json': 'application/json',
            'html': 'text/html',
            'htm': 'text/html',
            'xml': 'application/xml',
            'csv': 'text/csv',
            'md': 'text/markdown',
            'js': 'application/javascript',
            'css': 'text/css',
            'pdf': 'application/pdf',
            'jpg': 'image/jpeg',
            'jpeg': 'image/jpeg',
            'png': 'image/png',
            'gif': 'image/gif',
            'webp': 'image/webp',
            'svg': 'image/svg+xml',
            'mp3': 'audio/mpeg',
            'mp4': 'video/mp4',
            'zip': 'application/zip',
            'tar': 'application/x-tar',
            'gz': 'application/gzip'
        };
        return mimeTypes[ext] || 'application/octet-stream';
    },

    /**
     * Format bytes as human-readable size
     */
    formatBytes(bytes, decimals = 2) {
        if (bytes === 0) return '0 Bytes';
        const k = 1024;
        const dm = decimals < 0 ? 0 : decimals;
        const sizes = ['Bytes', 'KB', 'MB', 'GB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i];
    }
};
