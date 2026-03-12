# test-blog.ps1 -- smoke-test for the Forge Devlog example app.
#
# Run from the example/blog directory:
#   .\test-blog.ps1
#
# Starts blog.exe, waits for the server to accept connections, exercises every
# URL listed on the welcome page (plus a deliberate Draft -> 404 check), then
# stops the process.  Exits 0 on full success, 1 if any check fails.

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$baseUrl = "http://localhost:8080"
$exe     = ".\blog.exe"

if (-not (Test-Path $exe)) {
    Write-Error "blog.exe not found -- run 'go build .' first"
    exit 1
}

# ---------------------------------------------------------------------------
# Start the server
# ---------------------------------------------------------------------------
$app = Start-Process -FilePath $exe -PassThru -WindowStyle Hidden
Write-Host "Started blog.exe (PID $($app.Id))"

# ---------------------------------------------------------------------------
# Wait for the server to be ready (poll up to 10 s)
# ---------------------------------------------------------------------------
$ready   = $false
$elapsed = 0
while ($elapsed -lt 10) {
    try {
        $r = Invoke-WebRequest -Uri "$baseUrl/" -UseBasicParsing `
             -MaximumRedirection 0 -TimeoutSec 2 -ErrorAction Stop
        if ($r.StatusCode -eq 200) { $ready = $true; break }
    } catch { }
    Start-Sleep -Milliseconds 200
    $elapsed += 0.2
}

if (-not $ready) {
    Stop-Process -Id $app.Id -Force -ErrorAction SilentlyContinue
    Write-Error "Server did not become ready within 10 seconds"
    exit 1
}
Write-Host "Server ready.`n"

# ---------------------------------------------------------------------------
# Helper -- check one URL and return a result object
# ---------------------------------------------------------------------------
function Test-Url {
    param(
        [string]$method,
        [string]$url,
        [int]   $expect
    )

    $got = $null
    try {
        $r   = Invoke-WebRequest -Uri $url -Method $method -UseBasicParsing `
               -MaximumRedirection 0 -TimeoutSec 5 -ErrorAction SilentlyContinue
        $got = $r.StatusCode
    } catch {
        if ($_.Exception.Response) {
            $got = [int]$_.Exception.Response.StatusCode
        } else {
            $got = -1
        }
    }

    $pass   = $got -eq $expect
    $symbol = if ($pass) { "[PASS]" } else { "[FAIL]" }
    $label  = "$method $url"
    Write-Host ("  {0}  {1,-55}  expected={2}  got={3}" -f $symbol, $label, $expect, $got)
    return $pass
}

# ---------------------------------------------------------------------------
# URL checks
# ---------------------------------------------------------------------------
$results = @(
    (Test-Url "GET" "$baseUrl/"                                    200)
    (Test-Url "GET" "$baseUrl/posts"                               200)
    (Test-Url "GET" "$baseUrl/posts/forge-v1-release"              200)
    (Test-Url "GET" "$baseUrl/posts/forge-v1-release/aidoc"        200)
    (Test-Url "GET" "$baseUrl/posts/feed.xml"                      200)
    (Test-Url "GET" "$baseUrl/feed.xml"                            200)
    (Test-Url "GET" "$baseUrl/sitemap.xml"                         200)
    (Test-Url "GET" "$baseUrl/llms.txt"                            200)
    (Test-Url "GET" "$baseUrl/llms-full.txt"                       200)
    (Test-Url "GET" "$baseUrl/robots.txt"                          200)
    (Test-Url "GET" "$baseUrl/posts/mcp-support-preview"           404)
)

# ---------------------------------------------------------------------------
# Teardown
# ---------------------------------------------------------------------------
Stop-Process -Id $app.Id -Force -ErrorAction SilentlyContinue
Write-Host ""

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
$passed = ($results | Where-Object { $_ }).Count
$total  = $results.Count
$failed = $total - $passed

if ($failed -eq 0) {
    Write-Host "All $total checks passed." -ForegroundColor Green
    exit 0
} else {
    Write-Host "$failed of $total checks FAILED." -ForegroundColor Red
    exit 1
}
