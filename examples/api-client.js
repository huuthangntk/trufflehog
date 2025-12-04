#!/usr/bin/env node
/**
 * TruffleHog API Client Example (Node.js)
 * 
 * This script demonstrates how to use the TruffleHog REST API to scan repositories
 * for secrets and receive webhook notifications.
 */

const axios = require('axios');

class TruffleHogClient {
  constructor(baseUrl = 'http://localhost:8080') {
    this.baseUrl = baseUrl.replace(/\/$/, '');
    this.apiBase = `${this.baseUrl}/api/v1`;
  }

  async healthCheck() {
    try {
      const response = await axios.get(`${this.baseUrl}/health`);
      return response.status === 200;
    } catch (error) {
      return false;
    }
  }

  async initiateScan({ repoUrl, webhookUrl, verify = true, includeOnly }) {
    const payload = {
      repo_url: repoUrl,
      verify
    };

    if (webhookUrl) {
      payload.webhook_url = webhookUrl;
    }

    if (includeOnly) {
      payload.include_only = includeOnly;
    }

    const response = await axios.post(`${this.apiBase}/scan`, payload, {
      headers: { 'Content-Type': 'application/json' }
    });

    return response.data;
  }

  async getScanStatus(scanId) {
    const response = await axios.get(`${this.apiBase}/scan/status`, {
      params: { scan_id: scanId }
    });
    return response.data;
  }

  async listScans() {
    const response = await axios.get(`${this.apiBase}/scans`);
    return response.data;
  }

  async waitForScan(scanId, pollInterval = 5000, timeout = 600000) {
    const startTime = Date.now();

    while (true) {
      if (Date.now() - startTime > timeout) {
        throw new Error(`Scan ${scanId} did not complete within ${timeout / 1000} seconds`);
      }

      const result = await this.getScanStatus(scanId);
      const status = result.status;

      if (status === 'completed' || status === 'failed') {
        return result;
      }

      console.log(`Scan status: ${status}... waiting ${pollInterval / 1000}s`);
      await new Promise(resolve => setTimeout(resolve, pollInterval));
    }
  }
}

async function main() {
  const client = new TruffleHogClient('http://localhost:8080');

  // Check health
  const isHealthy = await client.healthCheck();
  if (!isHealthy) {
    console.log('❌ API server is not healthy');
    return;
  }
  console.log('✅ API server is healthy');

  // Initiate scan
  console.log('\n🔍 Initiating scan...');
  const scanResponse = await client.initiateScan({
    repoUrl: 'https://github.com/example/test-repo.git',
    webhookUrl: 'https://webhook.site/unique-id',
    verify: true,
    includeOnly: ['OpenAI', 'AWS', 'Perplexity', 'ElevenLabs']
  });

  const scanId = scanResponse.scan_id;
  console.log(`✅ Scan initiated: ${scanId}`);

  // Wait for completion
  console.log('\n⏳ Waiting for scan to complete...');
  try {
    const result = await client.waitForScan(scanId, 5000, 300000);

    console.log('\n' + '='.repeat(60));
    console.log(`Scan Status: ${result.status}`);
    console.log(`Repository: ${result.repo_url}`);
    console.log(`Started: ${result.started_at}`);
    console.log(`Completed: ${result.completed_at || 'N/A'}`);
    console.log('='.repeat(60));

    if (result.status === 'completed') {
      console.log('\n📊 Results:');
      console.log(`  Total Secrets: ${result.total_secrets}`);
      console.log(`  ✅ Verified: ${result.verified}`);
      console.log(`  ⚠️  Unverified: ${result.unverified}`);

      if (result.secrets && result.secrets.length > 0) {
        console.log('\n🔐 Found Secrets:');
        result.secrets.forEach((secret, i) => {
          const statusIcon = secret.verified ? '✅' : '⚠️';
          console.log(`\n  ${i + 1}. ${statusIcon} ${secret.detector_type}`);
          console.log(`     Redacted: ${secret.redacted}`);
          console.log(`     Source: ${secret.source_name}`);
          if (secret.extra_data) {
            console.log(`     Extra Data: ${JSON.stringify(secret.extra_data, null, 8)}`);
          }
        });
      }
    } else {
      console.log(`\n❌ Scan failed: ${result.error || 'Unknown error'}`);
    }
  } catch (error) {
    console.log(`\n❌ Error: ${error.message}`);
  }

  // List all scans
  console.log('\n' + '='.repeat(60));
  console.log('📋 All Scans:');
  console.log('='.repeat(60));
  const scans = await client.listScans();
  scans.scans.forEach(scan => {
    console.log(`\nScan ID: ${scan.scan_id}`);
    console.log(`  Status: ${scan.status}`);
    console.log(`  Repository: ${scan.repo_url}`);
    console.log(`  Secrets: ${scan.total_secrets || 0}`);
  });
}

main().catch(console.error);
