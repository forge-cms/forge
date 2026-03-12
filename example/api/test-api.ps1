# test-api.ps1 -- smoke-test for the Forge API example app.
#
# Run from the example/api directory:
#   .\test-api.ps1
#
# Starts api.exe, exercises every URL on the welcome page, then exercises all
# write operations via the built-in CLI (create/update/delete).
# Exits 0 on full success, 1 if any check fails.

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$baseUrl = "http://localhost:8082"
$exe     = ".\api.exe"

if (-not (Test-Path $exe)) {
    Write-Error "api.exe not found -- run 'go build .' first"
    exit 1
}

# ---------------------------------------------------------------------------
# Start the server
# ---------------------------------------------------------------------------
$app = Start-Process -FilePath $exe -PassThru -WindowStyle Hidden
Write-Host "Started api.exe (PID $($app.Id))"

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
# Helper -- check one URL and capture result
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
    Write-Host ("  {0}  {1,-67}  expected={2}  got={3}" -f $symbol, $label, $expect, $got)
    return $pass
}

# ---------------------------------------------------------------------------
# Helper -- POST with a JSON body and no auth header (expects 4xx)
# ---------------------------------------------------------------------------
function Test-UnauthWrite {
    param(
        [string]$url,
        [string]$body,
        [int[]] $expectAny    # accepts either 401 or 403
    )

    $got = $null
    try {
        $r   = Invoke-WebRequest -Uri $url -Method POST -Body $body `
               -ContentType "application/json" -UseBasicParsing `
               -MaximumRedirection 0 -TimeoutSec 5 -ErrorAction SilentlyContinue
        $got = $r.StatusCode
    } catch {
        if ($_.Exception.Response) {
            $got = [int]$_.Exception.Response.StatusCode
        } else {
            $got = -1
        }
    }

    $pass   = $expectAny -contains $got
    $symbol = if ($pass) { "[PASS]" } else { "[FAIL]" }
    Write-Host ("  {0}  POST $url (no auth)  expected={1}  got={2}" -f $symbol, ($expectAny -join "|"), $got)
    return $pass
}

# ---------------------------------------------------------------------------
# Read endpoint checks
# ---------------------------------------------------------------------------
Write-Host "  Read endpoints:"
$results = [System.Collections.Generic.List[bool]]::new()

$results.Add((Test-Url "GET" "$baseUrl/"                                    200))  # welcome page
$results.Add((Test-Url "GET" "$baseUrl/resources"                           200))  # list (public)
$results.Add((Test-Url "GET" "$baseUrl/resources/go-language-spec"          200))  # single resource (public)
$results.Add((Test-Url "GET" "$baseUrl/resources/go-spec"                   301))  # legacy redirect
$results.Add((Test-Url "GET" "$baseUrl/llms.txt"                            200))  # AI index
$results.Add((Test-Url "GET" "$baseUrl/llms-full.txt"                       200))  # AI corpus
$results.Add((Test-Url "GET" "$baseUrl/resources/sitemap.xml"               200))  # sitemap fragment
$results.Add((Test-Url "GET" "$baseUrl/resources/feed.xml"                  200))  # RSS feed
$results.Add((Test-Url "GET" "$baseUrl/.well-known/redirects.json"          200))  # redirect manifest
$results.Add((Test-Url "GET" "$baseUrl/robots.txt"                          200))  # robots

# ---------------------------------------------------------------------------
# Auth check -- unauthenticated write returns 403 (Forge returns ErrForbidden)
# ---------------------------------------------------------------------------
Write-Host ""
Write-Host "  Auth check:"
$unauthBody = '{"title":"Unauth Test","url":"https://example.com","description":"test without credentials"}'
$results.Add((Test-UnauthWrite "$baseUrl/resources" $unauthBody @(401, 403)))

# ---------------------------------------------------------------------------
# CLI write-operation checks
# ---------------------------------------------------------------------------
Write-Host ""
Write-Host "  CLI write operations:"

# -- create --
$createOut = (& $exe create "Smoke Test Resource" "https://smoke.example.com" "A temporary resource created by the smoke test." 2>$null) -join "`n"
$createOk  = $LASTEXITCODE -eq 0
$slug      = ""
if ($createOk) {
    try { $slug = ($createOut | ConvertFrom-Json).slug } catch { $createOk = $false }
}
$symbol = if ($createOk) { "[PASS]" } else { "[FAIL]" }
Write-Host ("  {0}  CLI: create `"Smoke Test Resource`"  ->  slug: {1}" -f $symbol, $slug)
$results.Add($createOk)

# -- get (newly created resource must be visible as Published) --
if ($slug -ne "") {
    $results.Add((Test-Url "GET" "$baseUrl/resources/$slug" 200))
} else {
    Write-Host "  [FAIL]  GET /resources/<slug>  (skipped: create failed)"
    $results.Add($false)
}

# -- update --
if ($slug -ne "") {
    $null = (& $exe update $slug "Smoke Test Updated" "https://smoke.example.com" "Updated description for the smoke test resource." 2>$null)
    $updateOk = $LASTEXITCODE -eq 0
    $symbol   = if ($updateOk) { "[PASS]" } else { "[FAIL]" }
    Write-Host ("  {0}  CLI: update $slug" -f $symbol)
    $results.Add($updateOk)
} else {
    Write-Host "  [FAIL]  CLI: update (skipped: create failed)"
    $results.Add($false)
}

# -- delete --
if ($slug -ne "") {
    $null = (& $exe delete $slug 2>$null)
    $deleteOk = $LASTEXITCODE -eq 0
    $symbol   = if ($deleteOk) { "[PASS]" } else { "[FAIL]" }
    Write-Host ("  {0}  CLI: delete $slug" -f $symbol)
    $results.Add($deleteOk)
} else {
    Write-Host "  [FAIL]  CLI: delete (skipped: create failed)"
    $results.Add($false)
}

# -- verify deleted (Draft or gone -> 404) --
if ($slug -ne "") {
    $results.Add((Test-Url "GET" "$baseUrl/resources/$slug" 404))
} else {
    Write-Host "  [FAIL]  verify deleted (skipped: create failed)"
    $results.Add($false)
}

# ---------------------------------------------------------------------------
# Teardown JSON server
# ---------------------------------------------------------------------------
Stop-Process -Id $app.Id -Force -ErrorAction SilentlyContinue
Write-Host ""

# ---------------------------------------------------------------------------
# HTML mode checks -- restart server with "html" flag
# ---------------------------------------------------------------------------
Write-Host "  HTML mode (./api html):"
$htmlApp = Start-Process -FilePath $exe -ArgumentList "html" -PassThru -WindowStyle Hidden

$htmlReady   = $false
$htmlElapsed = 0
while ($htmlElapsed -lt 10) {
    try {
        $r = Invoke-WebRequest -Uri "$baseUrl/" -UseBasicParsing `
             -MaximumRedirection 0 -TimeoutSec 2 -ErrorAction Stop
        if ($r.StatusCode -eq 200) { $htmlReady = $true; break }
    } catch { }
    Start-Sleep -Milliseconds 200
    $htmlElapsed += 0.2
}

if (-not $htmlReady) {
    Stop-Process -Id $htmlApp.Id -Force -ErrorAction SilentlyContinue
    Write-Host "  [FAIL]  HTML mode server did not start"
    $results.Add($false)
    $results.Add($false)
} else {
    # Accept header matching a real browser — triggers HTML rendering
    $browserAccept = "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"

    # list page returns text/html
    $htmlListOk = $false
    try {
        $r = Invoke-WebRequest -Uri "$baseUrl/resources" `
             -Headers @{Accept = $browserAccept} -UseBasicParsing `
             -TimeoutSec 5 -ErrorAction Stop
        $ct = $r.Headers["Content-Type"]
        $htmlListOk = ($ct -like "text/html*") -and ($r.Content -like "<!DOCTYPE html*" -or $r.Content -like "<!doctype html*")
    } catch { }
    $symbol = if ($htmlListOk) { "[PASS]" } else { "[FAIL]" }
    Write-Host ("  {0}  GET /resources  (text/html)" -f $symbol)
    $results.Add($htmlListOk)

    # show page returns text/html
    $htmlShowOk = $false
    try {
        $r = Invoke-WebRequest -Uri "$baseUrl/resources/go-language-spec" `
             -Headers @{Accept = $browserAccept} -UseBasicParsing `
             -TimeoutSec 5 -ErrorAction Stop
        $ct = $r.Headers["Content-Type"]
        $htmlShowOk = ($ct -like "text/html*") -and ($r.Content -like "*Visit resource*")
    } catch { }
    $symbol = if ($htmlShowOk) { "[PASS]" } else { "[FAIL]" }
    Write-Host ("  {0}  GET /resources/go-language-spec  (text/html)" -f $symbol)
    $results.Add($htmlShowOk)

    Stop-Process -Id $htmlApp.Id -Force -ErrorAction SilentlyContinue
}
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

