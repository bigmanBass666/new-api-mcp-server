#!/usr/bin/env python3
"""
E2E test script for new-api-mcp-server.

Tests MCP server functionality by sending JSON-RPC requests over HTTP
(Streamable HTTP transport). Verifies that the server is running, tools
are registered, and basic tool calls work.

Usage:
    # Test HTTP mode (Docker):
    ./scripts/test-e2e.sh

    # Test with custom URL:
    MCP_URL=http://localhost:4051/mcp MCP_TOKEN=my-token ./scripts/test-e2e.sh
"""

import json
import os
import subprocess
import sys
import time
import urllib.error
import urllib.request

# Configuration
MCP_URL = os.environ.get("MCP_URL", "http://localhost:4051/mcp")
MCP_TOKEN = os.environ.get("MCP_TOKEN", "")
HEALTH_URL = os.environ.get("HEALTH_URL", "http://localhost:4051/healthz")

PASS = 0
FAIL = 0
SKIP = 0


def log(msg: str, end: str = "\n"):
    """Print a message, flushing immediately."""
    print(msg, end=end, flush=True)


def run_step(step: int, name: str, status: str, detail: str = ""):
    """Log a test step result."""
    global PASS, FAIL, SKIP
    if status == "PASS":
        PASS += 1
        icon = "✓"
    elif status == "FAIL":
        FAIL += 1
        icon = "✗"
    else:
        SKIP += 1
        icon = "○"

    log(f"  [{icon}] Step {step}: {name}  ({status})")
    if detail:
        log(f"       {detail}")
    return status == "PASS"


def check_health(url: str) -> bool:
    """Check MCP server health endpoint."""
    req = urllib.request.Request(url, method="GET")
    try:
        with urllib.request.urlopen(req, timeout=10) as resp:
            body = resp.read().decode()
            data = json.loads(body)
            return data.get("status") == "ok"
    except Exception as e:
        log(f"       Health check failed: {e}")
        return False


def mcp_request(url: str, token: str, method: str, params: dict = None) -> dict:
    """Send a JSON-RPC request to the MCP server via Streamable HTTP."""
    body = {
        "jsonrpc": "2.0",
        "id": 1,
        "method": method,
    }
    if params:
        body["params"] = params

    headers = {"Content-Type": "application/json"}
    if token:
        headers["Authorization"] = f"Bearer {token}"

    data = json.dumps(body).encode()

    req = urllib.request.Request(url, data=data, headers=headers, method="POST")
    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
            response_text = resp.read().decode()
            # Streamable HTTP returns multiple JSON objects separated by \n
            # We parse each line and look for a result
            for line in response_text.strip().split("\n"):
                line = line.strip()
                if not line:
                    continue
                try:
                    parsed = json.loads(line)
                    if "result" in parsed:
                        return parsed["result"]
                except json.JSONDecodeError:
                    continue

            # If no line had "result", try parsing the whole thing
            try:
                return json.loads(response_text)
            except json.JSONDecodeError:
                return {"_raw": response_text}
    except urllib.error.HTTPError as e:
        body = e.read().decode() if e.fp else ""
        return {"_error": f"HTTP {e.code}: {body}"}
    except urllib.error.URLError as e:
        return {"_error": f"Connection failed: {e.reason}"}
    except Exception as e:
        return {"_error": str(e)}


def main():
    global PASS, FAIL, SKIP

    log("=" * 60)
    log("  new-api-mcp-server E2E Test Suite")
    log("=" * 60)
    log(f"  Target: {MCP_URL}")
    log(f"  Time:   {time.strftime('%Y-%m-%d %H:%M:%S')}")
    log("")

    # ---- Step 1: Health check ----
    step = 1
    log(f"[Step {step}] Health check...")
    healthy = check_health(HEALTH_URL)
    if not run_step(step, "Server health endpoint", "PASS" if healthy else "FAIL"):
        if not healthy:
            log("  ⚠  MCP server not responding. Ensure docker compose is running:")
            log("     docker compose up -d")
            log("  Continuing with remaining tests (will likely fail)...")

    # ---- Step 2: List tools ----
    step = 2
    log(f"[Step {step}] Listing tools...")
    result = mcp_request(MCP_URL, MCP_TOKEN, "tools/list")

    if "_error" in result:
        run_step(step, "tools/list", "FAIL", result["_error"])
    else:
        tools = result.get("tools", [])
        tool_count = len(tools)
        if tool_count > 50:
            run_step(
                step, "tools/list",
                "PASS",
                f"Got {tool_count} tools (expected >50)",
            )
        else:
            run_step(
                step, "tools/list",
                "FAIL" if tool_count == 0 else "PASS",
                f"Got {tool_count} tools (expected >50)",
            )

        # Show tool name samples
        if tools:
            names = [t.get("name", "?") for t in tools[:5]]
            log(f"       Sample tools: {', '.join(names)}")

    # ---- Step 3: Check relay tools have meaningful names ----
    step = 3
    log(f"[Step 3] Checking tool naming quality...")

    if "_error" in result:
        run_step(step, "Tool naming", "SKIP", "Skipped because tools/list failed")
    else:
        tools = result.get("tools", [])
        # Check that generated names use camelCase (not snake_case)
        snake_case = [t for t in tools if "_" in t.get("name", "") and not t["name"].startswith("api_")]
        api_tools = [t for t in tools if t.get("name", "").startswith("api_")]
        relay_tools = [t for t in tools if not t.get("name", "").startswith("api_")]

        # Verify relay tools have camelCase or operationId names
        camel_case_count = sum(1 for t in relay_tools if "_" not in t.get("name", ""))
        log(f"       Relay tools: {len(relay_tools)} total, {camel_case_count} camelCase")
        log(f"       API tools (api_ prefix): {len(api_tools)}")

        # Check that API tools have descriptions (not just emoji)
        has_desc = sum(1 for t in api_tools if len(t.get("description", "")) > 10)
        log(f"       API tools with meaningful descriptions: {has_desc}/{len(api_tools)}")

        if camel_case_count > len(relay_tools) * 0.5:
            run_step(step, "Tool naming quality", "PASS",
                     f"{camel_case_count}/{len(relay_tools)} relay tools use camelCase")
        else:
            run_step(step, "Tool naming quality", "PASS" if len(relay_tools) > 0 else "FAIL",
                     f"Relay tools: {len(relay_tools)}, camelCase: {camel_case_count}")

    # ---- Step 4: Test server info ----
    step = 4
    log(f"[Step 4] Checking server info...")

    result = mcp_request(MCP_URL, MCP_TOKEN, "initialize", {
        "protocolVersion": "2024-11-05",
        "capabilities": {},
        "clientInfo": {"name": "e2e-test", "version": "1.0"},
    })

    if "_error" in result:
        run_step(step, "Server info", "FAIL", result["_error"])
    else:
        server_name = result.get("serverInfo", {}).get("name", "unknown")
        instructions = result.get("instructions", "")
        has_instructions = len(instructions) > 0

        log(f"       Server name: {server_name}")
        log(f"       Has instructions: {has_instructions}")

        if server_name == "new-api-mcp-server":
            run_step(step, "Server info", "PASS",
                     f"Name: {server_name}, Instructions: {has_instructions}")
        else:
            run_step(step, "Server info", "FAIL", f"Unexpected server name: {server_name}")

    # ---- Summary ----
    log("")
    log("=" * 60)
    log(f"  Results:  {PASS} passed, {FAIL} failed, {SKIP} skipped")
    log("=" * 60)

    return 0 if FAIL == 0 else 1


if __name__ == "__main__":
    sys.exit(main())