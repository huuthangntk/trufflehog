#!/usr/bin/env node
/**
 * TruffleHog Webhook Server Example
 * 
 * This is a simple Express server that receives webhook notifications
 * from the TruffleHog API when scans complete.
 */

const express = require('express');
const bodyParser = require('body-parser');

const app = express();
const PORT = process.env.PORT || 3000;

// Middleware
app.use(bodyParser.json());

// Webhook endpoint
app.post('/webhook', (req, res) => {
  const { event, scan_result, timestamp } = req.body;
  
  console.log('\n' + '='.repeat(60));
  console.log('📬 Webhook Received');
  console.log('='.repeat(60));
  console.log(`Event: ${event}`);
  console.log(`Timestamp: ${timestamp}`);
  
  if (event === 'scan.completed') {
    console.log(`\nScan ID: ${scan_result.scan_id}`);
    console.log(`Status: ${scan_result.status}`);
    console.log(`Repository: ${scan_result.repo_url}`);
    console.log(`Duration: ${scan_result.started_at} → ${scan_result.completed_at}`);
    
    if (scan_result.status === 'completed') {
      console.log(`\n📊 Results:`);
      console.log(`  Total Secrets: ${scan_result.total_secrets}`);
      console.log(`  ✅ Verified: ${scan_result.verified}`);
      console.log(`  ⚠️  Unverified: ${scan_result.unverified}`);
      
      if (scan_result.secrets && scan_result.secrets.length > 0) {
        console.log(`\n🔐 Secrets Found:`);
        
        // Group by detector type
        const byType = {};
        scan_result.secrets.forEach(secret => {
          if (!byType[secret.detector_type]) {
            byType[secret.detector_type] = { verified: 0, unverified: 0 };
          }
          if (secret.verified) {
            byType[secret.detector_type].verified++;
          } else {
            byType[secret.detector_type].unverified++;
          }
        });
        
        Object.entries(byType).forEach(([type, counts]) => {
          console.log(`  ${type}:`);
          console.log(`    ✅ Verified: ${counts.verified}`);
          console.log(`    ⚠️  Unverified: ${counts.unverified}`);
        });
        
        // Show verified secrets (redacted)
        const verifiedSecrets = scan_result.secrets.filter(s => s.verified);
        if (verifiedSecrets.length > 0) {
          console.log(`\n⚠️  VERIFIED SECRETS (ACTION REQUIRED):`);
          verifiedSecrets.forEach((secret, i) => {
            console.log(`\n  ${i + 1}. ${secret.detector_type}`);
            console.log(`     Redacted: ${secret.redacted}`);
            console.log(`     Source: ${secret.source_name}`);
            if (secret.extra_data) {
              console.log(`     Details: ${JSON.stringify(secret.extra_data)}`);
            }
          });
        }
      }
      
      // Here you could:
      // - Send alerts to Slack/Discord
      // - Create JIRA tickets
      // - Send emails
      // - Update a dashboard
      // - Trigger remediation workflows
      
      if (scan_result.verified > 0) {
        console.log(`\n🚨 ALERT: ${scan_result.verified} verified secrets found!`);
        // sendSlackAlert(scan_result);
        // createJiraTicket(scan_result);
      }
      
    } else if (scan_result.status === 'failed') {
      console.log(`\n❌ Scan Failed: ${scan_result.error}`);
      // sendErrorNotification(scan_result);
    }
  }
  
  console.log('='.repeat(60) + '\n');
  
  // Always respond with 200 OK
  res.status(200).json({ received: true });
});

// Health check endpoint
app.get('/health', (req, res) => {
  res.json({ status: 'healthy' });
});

// Start server
app.listen(PORT, () => {
  console.log(`🎣 TruffleHog Webhook Server listening on port ${PORT}`);
  console.log(`Webhook URL: http://localhost:${PORT}/webhook`);
  console.log(`\nWaiting for webhook notifications...\n`);
});

// Example integration functions (implement as needed)

function sendSlackAlert(scanResult) {
  // Implementation for Slack webhook
  console.log('Sending Slack alert...');
}

function createJiraTicket(scanResult) {
  // Implementation for JIRA API
  console.log('Creating JIRA ticket...');
}

function sendErrorNotification(scanResult) {
  // Implementation for error notifications
  console.log('Sending error notification...');
}
