#!/usr/bin/env node

/**
 * Node.js helper script to configure proxy settings for Codex CLI
 * This ensures Node.js applications respect proxy environment variables
 */

const { spawn } = require('child_process');
const https = require('https');
const http = require('http');

// Configuration
const PROXY_HOST = process.env.PROXY_HOST || 'localhost';
const PROXY_PORT = process.env.PROXY_PORT || '8080';
const PROXY_URL = `http://${PROXY_HOST}:${PROXY_PORT}`;

console.log('🔧 Configuring Node.js proxy settings');
console.log('=====================================');
console.log(`Proxy URL: ${PROXY_URL}`);
console.log('');

// Set proxy environment variables
process.env.HTTP_PROXY = PROXY_URL;
process.env.HTTPS_PROXY = PROXY_URL;
process.env.http_proxy = PROXY_URL;
process.env.https_proxy = PROXY_URL;

// Optional: Set NO_PROXY for local addresses
process.env.NO_PROXY = process.env.NO_PROXY || 'localhost,127.0.0.1,::1';

// Configure global agents for Node.js HTTP/HTTPS modules
const HttpProxyAgent = require('http-proxy-agent');
const HttpsProxyAgent = require('https-proxy-agent');

// Note: These packages need to be installed:
// npm install http-proxy-agent https-proxy-agent

try {
    http.globalAgent = new HttpProxyAgent(PROXY_URL);
    https.globalAgent = new HttpsProxyAgent(PROXY_URL);
    console.log('✅ Global agents configured');
} catch (error) {
    console.log('⚠️  Could not configure global agents (packages may not be installed)');
    console.log('   Run: npm install -g http-proxy-agent https-proxy-agent');
}

// Test proxy connection
console.log('');
console.log('Testing proxy connection...');

const testUrl = 'http://httpbin.org/ip';
const proxyTest = http.get(testUrl, { 
    agent: new (require('http-proxy-agent'))(PROXY_URL) 
}, (res) => {
    let data = '';
    res.on('data', chunk => data += chunk);
    res.on('end', () => {
        try {
            const json = JSON.parse(data);
            console.log('✅ Proxy test successful!');
            console.log('   Your IP:', json.origin);
        } catch (e) {
            console.log('✅ Proxy connection works (response received)');
        }
    });
});

proxyTest.on('error', (err) => {
    console.log('❌ Proxy test failed:', err.message);
    console.log('   Make sure the proxy server is running');
});

// If arguments provided, execute command with proxy settings
const args = process.argv.slice(2);
if (args.length > 0) {
    console.log('');
    console.log('Executing command with proxy:', args.join(' '));
    console.log('');
    
    const child = spawn(args[0], args.slice(1), {
        stdio: 'inherit',
        env: process.env
    });
    
    child.on('exit', (code) => {
        process.exit(code);
    });
} else {
    console.log('');
    console.log('Usage:');
    console.log('  node setup-node-proxy.js <command> [args...]');
    console.log('');
    console.log('Example:');
    console.log('  node setup-node-proxy.js codex chat "Hello"');
    console.log('');
    console.log('Environment variables set:');
    console.log(`  HTTP_PROXY=${process.env.HTTP_PROXY}`);
    console.log(`  HTTPS_PROXY=${process.env.HTTPS_PROXY}`);
    console.log(`  NO_PROXY=${process.env.NO_PROXY}`);
}