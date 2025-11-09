#!/usr/bin/env python3
import os
import sys
import json
import time
import socket
import signal
import argparse
from datetime import datetime
from urllib.parse import urlparse, quote_plus
from urllib.request import Request, urlopen
from urllib.error import URLError, HTTPError
import concurrent.futures
import http.client
from pathlib import Path


# ANSI color codes
class Colors:
    RED = "\033[0;31m"
    GREEN = "\033[0;32m"
    YELLOW = "\033[1;33m"
    BLUE = "\033[0;34m"
    NC = "\033[0m"  # No Color


# Global variables
SCRIPT_DIR = Path(__file__).parent.absolute()
PROJECT_DIR = SCRIPT_DIR.parent
CONFIG_FILE = PROJECT_DIR / "presets" / "config.json"
REPORT_FILE = PROJECT_DIR / "node-report.md"
LAST_ERROR = ""
USER_AGENT = "Aqua-Speed-StatusChecker/1.0"
DEBUG = False


def log_info(msg):
    print(f"{Colors.BLUE}[INFO]{Colors.NC} {msg}")


def log_success(msg):
    print(f"{Colors.GREEN}[SUCCESS]{Colors.NC} {msg}")


def log_warning(msg):
    print(f"{Colors.YELLOW}[WARNING]{Colors.NC} {msg}")


def log_error(msg):
    print(f"{Colors.RED}[ERROR]{Colors.NC} {msg}")


def log_debug(msg):
    if DEBUG:
        print(f"{Colors.YELLOW}[DEBUG]{Colors.NC} {msg}")


def check_dependencies():
    """Placeholder for future external dependency checks."""
    return


def extract_hostname(url):
    hostname = urlparse(url).netloc
    log_debug(f"Extracted hostname '{hostname}' from URL '{url}'")
    return hostname


def resolve_dns(hostname):
    """Resolve DNS for hostname and return IP addresses"""
    global LAST_ERROR

    try:
        import socket

        log_debug(f"Starting DNS resolution for hostname: {hostname}")

        # Get address info
        addr_info = socket.getaddrinfo(hostname, None)
        ip_addresses = [str(addr[4][0]) for addr in addr_info]
        ip_addresses = list(set(ip_addresses))

        log_debug(f"DNS resolution successful for {hostname}:")
        for ip in ip_addresses:
            log_debug(f"  - {ip}")

        return ip_addresses
    except socket.gaierror as e:
        LAST_ERROR = f"DNS resolution failed: {e}"
        log_debug(f"DNS resolution failed for {hostname}: {e}")
        return []
    except Exception as e:
        LAST_ERROR = f"DNS resolution error: {e}"
        log_debug(f"DNS resolution error for {hostname}: {e}")
        return []


def call_check_host_api(url, timeout=8):
    global LAST_ERROR

    headers = {"Accept": "application/json", "User-Agent": USER_AGENT}
    request = Request(url, headers=headers)

    log_debug(f"Calling Check-Host API: {url}")

    try:
        with urlopen(request, timeout=timeout) as response:
            encoding = response.headers.get_content_charset("utf-8")
            raw = response.read().decode(encoding, errors="replace").strip()
            log_debug(f"Check-Host API raw response: {raw}")
            return json.loads(raw) if raw else {}
    except HTTPError as e:
        LAST_ERROR = f"Check-Host API HTTP {e.code}"
        body = e.read().decode("utf-8", errors="replace") if e.fp else ""
        log_debug(f"Check-Host API HTTP error {e.code}: {body}")
    except URLError as e:
        LAST_ERROR = f"Check-Host API network error: {e.reason}"
        log_debug(f"Check-Host API network error: {e}")
    except json.JSONDecodeError as e:
        LAST_ERROR = "Invalid JSON from Check-Host API"
        log_debug(f"Failed to decode JSON from Check-Host API: {e}")
    except Exception as e:
        LAST_ERROR = "Unexpected Check-Host API error"
        log_debug(f"Unexpected error when calling Check-Host API: {e}")

    return None


def extract_ping_statuses(node_payload):
    """Extract status strings (OK/TIMEOUT/...) from nested ping payload."""

    statuses = []

    def _walk(item):
        if isinstance(item, list):
            if item and isinstance(item[0], str):
                statuses.append(item[0])
            for sub in item:
                _walk(sub)

    _walk(node_payload)
    return statuses


def test_icmp_ping(hostname):
    global LAST_ERROR

    if DEBUG:
        ip_addresses = resolve_dns(hostname)
        if ip_addresses:
            log_debug(f"Local DNS resolved {hostname} -> {', '.join(ip_addresses)}")
        else:
            log_debug(
                f"Local DNS resolution failed for {hostname}, relying on Check-Host"
            )

    encoded_host = quote_plus(hostname)
    request_url = f"https://check-host.net/check-ping?host={encoded_host}&max_nodes=3"

    init_response = call_check_host_api(request_url)
    if not init_response:
        if not LAST_ERROR:
            LAST_ERROR = "Failed to start Check-Host ping request"
        return "❌ FAIL"

    if not init_response.get("ok"):
        LAST_ERROR = init_response.get("error", "Check-Host API returned ok=0")
        log_debug(f"Check-Host API returned failure: {init_response}")
        return "❌ FAIL"

    request_id = init_response.get("request_id")
    nodes = init_response.get("nodes", {})
    log_debug(
        f"Check-Host request_id={request_id}, nodes={', '.join(nodes.keys()) or 'none'}"
    )

    if not request_id:
        LAST_ERROR = "Check-Host API did not provide request_id"
        return "❌ FAIL"

    result_url = f"https://check-host.net/check-result/{request_id}"
    max_attempts = 6
    poll_result = None

    for attempt in range(1, max_attempts + 1):
        log_debug(f"Polling Check-Host result ({attempt}/{max_attempts})")
        time.sleep(1)
        poll_result = call_check_host_api(result_url)
        if poll_result is None:
            continue
        if not isinstance(poll_result, dict):
            log_debug("Unexpected Check-Host result payload, retrying...")
            poll_result = None
            continue

        if any(payload for payload in poll_result.values()):
            break
        log_debug("Check-Host still collecting results, waiting...")
        poll_result = None

    if poll_result is None:
        LAST_ERROR = "Check-Host result polling timeout"
        log_debug("Unable to obtain Check-Host results before timeout")
        return "❌ FAIL"

    total_nodes = len(nodes) or len(poll_result) or 1
    success_nodes = 0
    failure_details = []

    for node_name, payload in poll_result.items():
        statuses = extract_ping_statuses(payload)
        if statuses:
            log_debug(f"{node_name} statuses: {', '.join(statuses)}")
        else:
            log_debug(f"{node_name} returned empty payload: {payload}")

        if any(status.upper() == "OK" for status in statuses):
            success_nodes += 1
        elif payload is None:
            failure_details.append(f"{node_name}: no data returned")
        elif statuses:
            failure_details.append(f"{node_name}: {statuses[0]}")
        else:
            failure_details.append(f"{node_name}: pending")

    if success_nodes:
        return f"✅ PASS ({success_nodes}/{total_nodes} nodes OK)"

    LAST_ERROR = failure_details[0] if failure_details else "No Check-Host nodes reported OK"
    return "❌ FAIL"


def test_tcp_ping(hostname, port):
    global LAST_ERROR

    # Resolve DNS in debug mode
    if DEBUG:
        ip_addresses = resolve_dns(hostname)
        if not ip_addresses:
            log_debug(f"Skipping TCP ping due to DNS resolution failure")
            return "❌ FAIL"
        else:
            log_debug(
                f"Will test TCP connection to {hostname}:{port} -> {', '.join(ip_addresses)}"
            )

    try:
        log_debug(f"Creating TCP connection to {hostname}:{port} with 3s timeout")
        start_time = time.time()

        sock = socket.create_connection((hostname, port), timeout=3)

        end_time = time.time()
        connection_time = end_time - start_time

        # Get socket info for debug
        local_addr = sock.getsockname()
        peer_addr = sock.getpeername()

        log_debug(f"TCP connection established in {connection_time:.3f}s")
        log_debug(f"Local address: {local_addr[0]}:{local_addr[1]}")
        log_debug(f"Peer address: {peer_addr[0]}:{peer_addr[1]}")

        sock.close()
        log_debug(f"TCP connection closed successfully")
        return "✅ PASS"
    except socket.gaierror as e:
        LAST_ERROR = "DNS resolution failed"
        log_debug(f"TCP ping DNS resolution failed: {e}")
    except ConnectionRefusedError as e:
        LAST_ERROR = f"Port {port} closed or filtered"
        log_debug(f"TCP connection refused for {hostname}:{port} - {e}")
    except socket.timeout as e:
        LAST_ERROR = f"Port {port} connection timeout (3s)"
        log_debug(f"TCP connection timeout for {hostname}:{port} after 3s - {e}")
    except Exception as e:
        LAST_ERROR = f"Port {port} connection failed"
        log_debug(f"TCP connection failed for {hostname}:{port} - {e}")

    return "❌ FAIL"


def test_http_get(url):
    global LAST_ERROR

    try:
        parsed = urlparse(url)
        hostname = parsed.netloc
        path = parsed.path or "/"

        # Resolve DNS in debug mode
        if DEBUG:
            ip_addresses = resolve_dns(hostname)
            if not ip_addresses:
                log_debug(f"Skipping HTTP GET due to DNS resolution failure")
                return "❌ FAIL"
            else:
                log_debug(
                    f"Will make HTTP request to {hostname} -> {', '.join(ip_addresses)}"
                )

        log_debug(
            f"Creating HTTP{'S' if parsed.scheme == 'https' else ''} connection to {hostname}"
        )
        log_debug(f"Request URL: {url}")
        log_debug(f"Request path: {path}")

        conn = (
            http.client.HTTPSConnection(parsed.netloc, timeout=5)
            if parsed.scheme == "https"
            else http.client.HTTPConnection(parsed.netloc, timeout=5)
        )

        headers = {"User-Agent": USER_AGENT}
        log_debug(f"Request headers: {headers}")

        start_time = time.time()
        conn.request("GET", path, headers=headers)

        response = conn.getresponse()
        response_time = time.time() - start_time

        log_debug(f"HTTP response received in {response_time:.3f}s")
        log_debug(f"Response status: {response.status} {response.reason}")
        log_debug(f"Response headers: {dict(response.getheaders())}")

        # Read some data and log details
        data = response.read(1024)
        log_debug(f"Read {len(data)} bytes of response data")
        if len(data) == 1024:
            log_debug("Response has more data available (read limit reached)")

        content_type = response.getheader("content-type", "unknown")
        content_length = response.getheader("content-length", "unknown")
        log_debug(f"Content-Type: {content_type}")
        log_debug(f"Content-Length: {content_length}")

        conn.close()
        log_debug("HTTP connection closed successfully")
        return "✅ PASS"
    except Exception as e:
        LAST_ERROR = "HTTP request failed"
        error_str = str(e)

        log_debug(f"HTTP request failed with error: {e}")
        log_debug(f"Error type: {type(e).__name__}")

        if "timeout" in error_str.lower():
            LAST_ERROR = "HTTP request timeout (5s)"
        elif "name resolution" in error_str.lower():
            LAST_ERROR = "DNS resolution failed"
        elif "connection refused" in error_str.lower():
            LAST_ERROR = "HTTP connection refused"
        elif "ssl" in error_str.lower():
            LAST_ERROR = "SSL/TLS connection error"

        return "❌ FAIL"


def test_single_thread(url, thread_id=None):
    try:
        thread_prefix = f"[Thread-{thread_id}] " if thread_id is not None else ""
        log_debug(f"{thread_prefix}Starting HTTP request to {url}")

        parsed = urlparse(url)
        conn = (
            http.client.HTTPSConnection(parsed.netloc, timeout=2)
            if parsed.scheme == "https"
            else http.client.HTTPConnection(parsed.netloc, timeout=2)
        )
        headers = {"User-Agent": USER_AGENT}

        start_time = time.time()
        conn.request("GET", parsed.path or "/", headers=headers)
        response = conn.getresponse()
        data = response.read(1024)
        conn.close()

        elapsed = time.time() - start_time
        log_debug(
            f"{thread_prefix}Success - {response.status} {response.reason} in {elapsed:.3f}s, read {len(data)} bytes"
        )
        return True
    except Exception as e:
        thread_prefix = f"[Thread-{thread_id}] " if thread_id is not None else ""
        log_debug(f"{thread_prefix}Failed - {type(e).__name__}: {e}")
        return False


def test_multithreaded_get(url):
    global LAST_ERROR
    threads = 8
    success_count = 0

    log_debug(f"Starting multi-threaded test with {threads} threads for {url}")

    # Resolve DNS once in debug mode
    if DEBUG:
        hostname = extract_hostname(url)
        ip_addresses = resolve_dns(hostname)
        if not ip_addresses:
            log_debug(f"Skipping multi-threaded test due to DNS resolution failure")
            return "❌ FAIL"
        else:
            log_debug(f"Multi-threaded test will connect to: {', '.join(ip_addresses)}")

    with concurrent.futures.ThreadPoolExecutor(max_workers=threads) as executor:
        # Set a 10 second timeout for the entire multi-threaded test
        try:
            start_time = time.time()
            futures = [
                executor.submit(test_single_thread, url, i + 1) for i in range(threads)
            ]
            results = []

            for i, future in enumerate(
                concurrent.futures.as_completed(futures, timeout=10)
            ):
                result = future.result()
                results.append(result)
                log_debug(
                    f"Thread {i+1} completed: {'SUCCESS' if result else 'FAILED'}"
                )

            success_count = sum(results)
            elapsed = time.time() - start_time

            log_debug(f"Multi-threaded test completed in {elapsed:.3f}s")
            log_debug(f"Results: {success_count}/{threads} threads successful")

        except concurrent.futures.TimeoutError:
            elapsed = time.time() - start_time
            LAST_ERROR = "Multi-thread test timeout (10s)"
            log_debug(f"Multi-threaded test timeout after {elapsed:.3f}s")
            return f"❌ FAIL (timeout)"

    if success_count >= 6:  # At least 75% success
        log_debug(
            f"Multi-threaded test passed with {success_count}/{threads} successful threads"
        )
        return f"✅ PASS ({success_count}/{threads})"
    else:
        LAST_ERROR = f"Multi-thread test failed: only {success_count}/{threads} threads succeeded"
        log_debug(
            f"Multi-threaded test failed - insufficient success rate: {success_count}/{threads}"
        )
        return f"❌ FAIL ({success_count}/{threads})"


def init_report():
    with open(REPORT_FILE, "w", encoding="utf-8") as f:
        f.write(
            f"""# Node Health Status Report

Generated on: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}

## Configuration File Analysis

Configuration file: `presets/config.json`

## Test Results

| ID | Node Name | ISP | Type | ICMP Ping | TCP Ping | HTTP GET | 8-Thread GET | Notes |
|----|-----------|-----|------|-----------|----------|----------|--------------|-------|
"""
        )


def add_report_line(id, name, isp, type, icmp, tcp, http, multithread, notes):
    with open(REPORT_FILE, "a", encoding="utf-8") as f:
        f.write(
            f"| {id} | {name} | {isp} | {type} | {icmp} | {tcp} | {http} | {multithread} | {notes} |\n"
        )


def timeout_handler(signum, frame):
    raise TimeoutError("Node test timeout")


def test_node_with_timeout(id, name, url, isp, node_type, timeout=60):
    global LAST_ERROR

    old_handler = signal.signal(signal.SIGALRM, timeout_handler)
    signal.alarm(timeout)

    try:
        return test_node(id, name, url, isp, node_type)
    except TimeoutError:
        log_warning(f"Node test timeout after {timeout}s, skipping remaining tests")
        add_report_line(
            id,
            name,
            isp,
            node_type,
            "❌ TIMEOUT",
            "❌ TIMEOUT",
            "❌ TIMEOUT",
            "❌ TIMEOUT",
            f"Node test timeout after {timeout}s",
        )
        return 0
    finally:
        signal.alarm(0)  # Cancel the alarm
        signal.signal(signal.SIGALRM, old_handler)  # Restore old handler


def test_node(id, name, url, isp, node_type):
    hostname = extract_hostname(url)
    notes_array = []

    log_debug(f"=== Starting tests for node {id} ===")
    log_debug(f"Node details: {name} | {isp} | {node_type}")
    log_debug(f"Target URL: {url}")
    log_debug(f"Target hostname: {hostname}")

    print("    ICMP Ping: ", end="", flush=True)
    global LAST_ERROR
    LAST_ERROR = ""
    log_debug("--- Starting ICMP Ping test ---")
    icmp_result = test_icmp_ping(hostname)
    print(icmp_result)
    log_debug(f"ICMP Ping result: {icmp_result}")
    if "FAIL" in icmp_result and LAST_ERROR:
        notes_array.append(f"ICMP: {LAST_ERROR}")
        log_debug(f"ICMP Ping error: {LAST_ERROR}")

    port = "443" if url.startswith("https://") else "80"
    print(f"    TCP Ping ({port}): ", end="", flush=True)
    LAST_ERROR = ""
    log_debug(f"--- Starting TCP Ping test (port {port}) ---")
    tcp_result = test_tcp_ping(hostname, int(port))
    print(tcp_result)
    log_debug(f"TCP Ping result: {tcp_result}")
    if "FAIL" in tcp_result and LAST_ERROR:
        notes_array.append(f"TCP: {LAST_ERROR}")
        log_debug(f"TCP Ping error: {LAST_ERROR}")

    print("    HTTP GET: ", end="", flush=True)
    LAST_ERROR = ""
    log_debug("--- Starting HTTP GET test ---")
    http_result = test_http_get(url)
    print(http_result)
    log_debug(f"HTTP GET result: {http_result}")
    if "FAIL" in http_result and LAST_ERROR:
        notes_array.append(f"HTTP: {LAST_ERROR}")
        log_debug(f"HTTP GET error: {LAST_ERROR}")

    print("    8-Thread GET: ", end="", flush=True)
    LAST_ERROR = ""
    log_debug("--- Starting 8-Thread GET test ---")
    multi_result = test_multithreaded_get(url)
    print(multi_result)
    log_debug(f"8-Thread GET result: {multi_result}")
    if "FAIL" in multi_result and LAST_ERROR:
        notes_array.append(f"Multi: {LAST_ERROR}")
        log_debug(f"8-Thread GET error: {LAST_ERROR}")

    notes = "; ".join(notes_array) if notes_array else "All tests passed"
    log_debug(f"Final notes: {notes}")

    add_report_line(
        id,
        name,
        isp,
        node_type,
        icmp_result,
        tcp_result,
        http_result,
        multi_result,
        notes,
    )

    passed = sum(
        1
        for result in [icmp_result, tcp_result, http_result, multi_result]
        if "PASS" in result
    )
    log_debug(f"=== Node {id} testing completed: {passed}/4 tests passed ===")
    return passed


def run_tests():
    with open(CONFIG_FILE, "r", encoding="utf-8") as f:
        config = json.load(f)

    total_tests = 0
    passed_tests = 0
    node_count = len(config)

    log_info("Starting node health status checks...")

    for node_id, node_data in config.items():
        node_count += 1

        name_zh = node_data["name"]["zh"]
        name_en = node_data["name"]["en"]
        name = f"{name_zh} ({name_en})"
        isp_zh = node_data["isp"]["zh"]
        isp_en = node_data["isp"]["en"]
        isp = f"{isp_zh} ({isp_en})"
        url = node_data["url"]
        node_type = node_data["type"]
        size = node_data["size"]
        threads = node_data["threads"]
        country = node_data.get("geoInfo", {}).get("countryCode", "N/A")
        region = node_data.get("geoInfo", {}).get("region", "N/A")
        city = node_data.get("geoInfo", {}).get("city", "N/A")

        log_info(f"Testing Node [{node_count}]: {node_id}")
        log_info(f"  Name: {name}")
        log_info(f"  ISP: {isp}")
        log_info(f"  Type: {node_type}")
        log_info(f"  URL: {url}")
        log_info(f"  Size: {size}MB, Threads: {threads}")
        log_info(f"  Location: {country}/{region}/{city}")

        passed = test_node_with_timeout(node_id, name, url, isp, node_type, timeout=60)
        passed_tests += passed
        total_tests += 4

        print()  # Empty line for separation

    # Add statistics to report
    success_rate = (passed_tests * 100) // total_tests if total_tests > 0 else 0

    with open(REPORT_FILE, "a", encoding="utf-8") as f:
        f.write(
            f"""
## Statistics

- Total Nodes: {node_count}
- Total Tests: {total_tests}
- Passed: {passed_tests}
- Failed: {total_tests - passed_tests}
- Success Rate: {success_rate}%

## Health Status

"""
        )

        if success_rate >= 90:
            f.write(f"🟢 **HEALTHY** - Success rate: {success_rate}%\n")
        elif success_rate >= 70:
            f.write(f"🟡 **WARNING** - Success rate: {success_rate}%\n")
        else:
            f.write(f"🔴 **CRITICAL** - Success rate: {success_rate}%\n")

    log_success("Node health check completed!")
    log_info(f"Total nodes: {node_count}")
    log_info(
        f"Total tests: {total_tests}, Passed: {passed_tests}, Failed: {total_tests - passed_tests}"
    )
    log_info(f"Success rate: {success_rate}%")

    if success_rate >= 90:
        log_success("Overall health status: HEALTHY 🟢")
    elif success_rate >= 70:
        log_warning("Overall health status: WARNING 🟡")
    else:
        log_error("Overall health status: CRITICAL 🔴")

    log_info(f"Report saved to: {REPORT_FILE}")


def main():
    global DEBUG
    parser = argparse.ArgumentParser(
        description="Node Health Status Checker for Aqua Speed Tools"
    )
    parser.add_argument("--debug", action="store_true", help="Enable debug output")
    args = parser.parse_args()

    if args.debug:
        DEBUG = True
        log_info("Debug mode enabled.")

    log_info("Node Health Status Checker - Aqua Speed Tools")
    log_info(f"Config file: {CONFIG_FILE}")
    log_info(f"Report file: {REPORT_FILE}")

    if not CONFIG_FILE.exists():
        log_error(f"Configuration file not found: {CONFIG_FILE}")
        sys.exit(1)

    check_dependencies()

    try:
        with open(CONFIG_FILE, "r") as f:
            json.load(f)
    except json.JSONDecodeError:
        log_error("Invalid JSON format in configuration file")
        sys.exit(1)

    init_report()
    run_tests()


if __name__ == "__main__":
    main()
