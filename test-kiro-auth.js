#!/usr/bin/env node

// 测试 Kiro Auth 认证流程
const https = require('https');
const http = require('http');

const REGION = 'us-east-1';
const SSO_OIDC_ENDPOINT = `https://oidc.${REGION}.amazonaws.com`;
const USER_AGENT = 'Kiro IDE/1.0';

async function makeRequest(url, options = {}) {
  return new Promise((resolve, reject) => {
    const req = https.request(url, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'User-Agent': USER_AGENT,
        ...options.headers
      }
    }, (res) => {
      let data = '';
      res.on('data', chunk => data += chunk);
      res.on('end', () => {
        try {
          const parsed = JSON.parse(data);
          resolve({ status: res.statusCode, data: parsed });
        } catch (e) {
          resolve({ status: res.statusCode, data: data });
        }
      });
    });
    
    req.on('error', reject);
    
    if (options.body) {
      req.write(JSON.stringify(options.body));
    }
    req.end();
  });
}

async function testKiroAuth() {
  console.log('🔍 测试 Kiro Auth 认证流程...\n');
  
  try {
    // 步骤 1: 注册客户端
    console.log('1️⃣ 注册 OAuth 客户端...');
    const registerResult = await makeRequest(`${SSO_OIDC_ENDPOINT}/client/register`, {
      body: {
        clientName: 'Kiro IDE',
        clientType: 'public',
        scopes: ['sso:account:access'],
        grantTypes: ['urn:ietf:params:oauth:grant-type:device_code', 'refresh_token']
      }
    });
    
    if (registerResult.status !== 200) {
      console.error('❌ 客户端注册失败:', registerResult.status, registerResult.data);
      return;
    }
    
    const { clientId, clientSecret } = registerResult.data;
    console.log('✅ 客户端注册成功');
    console.log(`   Client ID: ${clientId.substring(0, 10)}...`);
    
    // 步骤 2: 获取设备授权码
    console.log('\n2️⃣ 获取设备授权码...');
    const deviceAuthResult = await makeRequest(`${SSO_OIDC_ENDPOINT}/device_authorization`, {
      body: {
        clientId,
        clientSecret,
        startUrl: 'https://view.awsapps.com/start'
      }
    });
    
    if (deviceAuthResult.status !== 200) {
      console.error('❌ 设备授权失败:', deviceAuthResult.status, deviceAuthResult.data);
      return;
    }
    
    const { verificationUri, verificationUriComplete, userCode, deviceCode, interval = 5 } = deviceAuthResult.data;
    console.log('✅ 设备授权成功');
    console.log(`   验证 URL: ${verificationUri}`);
    console.log(`   用户码: ${userCode}`);
    console.log(`   完整 URL: ${verificationUriComplete}`);
    
    // 步骤 3: 启动本地服务器显示认证页面
    console.log('\n3️⃣ 启动本地认证服务器...');
    const server = http.createServer((req, res) => {
      if (req.url === '/') {
        res.writeHead(200, { 'Content-Type': 'text/html' });
        res.end(`
          <html>
            <head><title>Kiro Auth Test</title></head>
            <body style="font-family: Arial; text-align: center; padding: 50px;">
              <h1>AWS Builder ID 认证测试</h1>
              <p>请在浏览器中访问: <a href="${verificationUriComplete}" target="_blank">${verificationUri}</a></p>
              <p>输入验证码: <strong>${userCode}</strong></p>
              <p>完成认证后，此页面会自动更新</p>
            </body>
          </html>
        `);
      } else {
        res.writeHead(404);
        res.end();
      }
    });
    
    server.listen(19847, '127.0.0.1', () => {
      console.log('✅ 认证服务器启动成功: http://127.0.0.1:19847');
      console.log('\n🌐 请在浏览器中完成认证...');
      console.log(`   访问: ${verificationUriComplete}`);
      console.log(`   输入码: ${userCode}`);
    });
    
    // 步骤 4: 轮询 token
    console.log('\n4️⃣ 开始轮询 token...');
    let attempts = 0;
    const maxAttempts = 20; // 最多尝试 20 次
    
    const pollToken = async () => {
      attempts++;
      console.log(`   尝试 ${attempts}/${maxAttempts}...`);
      
      try {
        const tokenResult = await makeRequest(`${SSO_OIDC_ENDPOINT}/token`, {
          body: {
            clientId,
            clientSecret,
            deviceCode,
            grantType: 'urn:ietf:params:oauth:grant-type:device_code'
          }
        });
        
        console.log(`   响应状态: ${tokenResult.status}`);
        console.log(`   响应数据:`, tokenResult.data);
        
        if (tokenResult.status === 200 && tokenResult.data.accessToken) {
          console.log('\n🎉 认证成功！');
          console.log(`   Access Token: ${tokenResult.data.accessToken.substring(0, 20)}...`);
          server.close();
          return;
        }
        
        if (tokenResult.data.error) {
          const error = tokenResult.data.error;
          if (error === 'authorization_pending') {
            console.log('   ⏳ 等待用户完成认证...');
            if (attempts < maxAttempts) {
              setTimeout(pollToken, interval * 1000);
            } else {
              console.log('\n❌ 认证超时');
              server.close();
            }
            return;
          } else if (error === 'slow_down') {
            console.log('   🐌 请求过快，延长间隔...');
            setTimeout(pollToken, (interval + 5) * 1000);
            return;
          } else {
            console.error(`\n❌ 认证失败: ${error}`);
            console.error(`   错误描述: ${tokenResult.data.error_description || '无'}`);
            server.close();
            return;
          }
        }
        
        console.error('\n❌ 未知响应格式');
        server.close();
        
      } catch (error) {
        console.error(`\n❌ 轮询错误: ${error.message}`);
        server.close();
      }
    };
    
    // 开始轮询
    setTimeout(pollToken, interval * 1000);
    
  } catch (error) {
    console.error('❌ 测试失败:', error.message);
  }
}

testKiroAuth();