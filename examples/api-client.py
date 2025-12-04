#!/usr/bin/env python3
"""
TruffleHog API Client Example

This script demonstrates how to use the TruffleHog REST API to scan repositories
for secrets and receive webhook notifications.
"""

import requests
import time
import json
from typing import Dict, Optional

class TruffleHogClient:
    def __init__(self, base_url: str = "http://localhost:8080"):
        self.base_url = base_url.rstrip('/')
        self.api_base = f"{self.base_url}/api/v1"
    
    def health_check(self) -> bool:
        """Check if the API server is healthy."""
        try:
            response = requests.get(f"{self.base_url}/health")
            return response.status_code == 200
        except requests.RequestException:
            return False
    
    def initiate_scan(
        self,
        repo_url: str,
        webhook_url: Optional[str] = None,
        verify: bool = True,
        include_only: Optional[list] = None
    ) -> Dict:
        """
        Initiate a new repository scan.
        
        Args:
            repo_url: Git repository URL to scan
            webhook_url: Optional webhook URL for notifications
            verify: Whether to verify found secrets
            include_only: Optional list of detector names to use
        
        Returns:
            Dictionary with scan_id and status
        """
        payload = {
            "repo_url": repo_url,
            "verify": verify
        }
        
        if webhook_url:
            payload["webhook_url"] = webhook_url
        
        if include_only:
            payload["include_only"] = include_only
        
        response = requests.post(
            f"{self.api_base}/scan",
            json=payload,
            headers={"Content-Type": "application/json"}
        )
        response.raise_for_status()
        return response.json()
    
    def get_scan_status(self, scan_id: str) -> Dict:
        """
        Get the status and results of a scan.
        
        Args:
            scan_id: The scan ID to query
        
        Returns:
            Dictionary with scan status and results
        """
        response = requests.get(
            f"{self.api_base}/scan/status",
            params={"scan_id": scan_id}
        )
        response.raise_for_status()
        return response.json()
    
    def list_scans(self) -> Dict:
        """
        List all scans.
        
        Returns:
            Dictionary with list of scans
        """
        response = requests.get(f"{self.api_base}/scans")
        response.raise_for_status()
        return response.json()
    
    def wait_for_scan(
        self,
        scan_id: str,
        poll_interval: int = 5,
        timeout: int = 600
    ) -> Dict:
        """
        Wait for a scan to complete.
        
        Args:
            scan_id: The scan ID to wait for
            poll_interval: Seconds between status checks
            timeout: Maximum seconds to wait
        
        Returns:
            Final scan result
        """
        start_time = time.time()
        
        while True:
            if time.time() - start_time > timeout:
                raise TimeoutError(f"Scan {scan_id} did not complete within {timeout} seconds")
            
            result = self.get_scan_status(scan_id)
            status = result.get("status")
            
            if status in ["completed", "failed"]:
                return result
            
            print(f"Scan status: {status}... waiting {poll_interval}s")
            time.sleep(poll_interval)


def main():
    # Initialize client
    client = TruffleHogClient("http://localhost:8080")
    
    # Check health
    if not client.health_check():
        print("❌ API server is not healthy")
        return
    
    print("✅ API server is healthy")
    
    # Initiate scan
    print("\n🔍 Initiating scan...")
    scan_response = client.initiate_scan(
        repo_url="https://github.com/example/test-repo.git",
        webhook_url="https://webhook.site/unique-id",
        verify=True,
        include_only=["OpenAI", "AWS", "Perplexity", "ElevenLabs"]
    )
    
    scan_id = scan_response["scan_id"]
    print(f"✅ Scan initiated: {scan_id}")
    
    # Wait for completion
    print("\n⏳ Waiting for scan to complete...")
    try:
        result = client.wait_for_scan(scan_id, poll_interval=5, timeout=300)
        
        print(f"\n{'='*60}")
        print(f"Scan Status: {result['status']}")
        print(f"Repository: {result['repo_url']}")
        print(f"Started: {result['started_at']}")
        print(f"Completed: {result.get('completed_at', 'N/A')}")
        print(f"{'='*60}")
        
        if result['status'] == 'completed':
            print(f"\n📊 Results:")
            print(f"  Total Secrets: {result['total_secrets']}")
            print(f"  ✅ Verified: {result['verified']}")
            print(f"  ⚠️  Unverified: {result['unverified']}")
            
            if result.get('secrets'):
                print(f"\n🔐 Found Secrets:")
                for i, secret in enumerate(result['secrets'], 1):
                    status_icon = "✅" if secret['verified'] else "⚠️"
                    print(f"\n  {i}. {status_icon} {secret['detector_type']}")
                    print(f"     Redacted: {secret['redacted']}")
                    print(f"     Source: {secret['source_name']}")
                    if secret.get('extra_data'):
                        print(f"     Extra Data: {json.dumps(secret['extra_data'], indent=8)}")
        else:
            print(f"\n❌ Scan failed: {result.get('error', 'Unknown error')}")
    
    except TimeoutError as e:
        print(f"\n❌ {e}")
    except Exception as e:
        print(f"\n❌ Error: {e}")
    
    # List all scans
    print(f"\n{'='*60}")
    print("📋 All Scans:")
    print(f"{'='*60}")
    scans = client.list_scans()
    for scan in scans['scans']:
        print(f"\nScan ID: {scan['scan_id']}")
        print(f"  Status: {scan['status']}")
        print(f"  Repository: {scan['repo_url']}")
        print(f"  Secrets: {scan.get('total_secrets', 0)}")


if __name__ == "__main__":
    main()
