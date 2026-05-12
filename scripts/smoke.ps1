param(
    [string]$BaseUrl = "http://localhost:8080",
    [string]$ApiKey = "qk_demo_local_dev_key"
)

$ErrorActionPreference = "Stop"

function Assert-True {
    param(
        [bool]$Condition,
        [string]$Message
    )
    if (-not $Condition) {
        throw "Assertion failed: $Message"
    }
}

function Get-RedirectHeaders {
    param([string]$Url)
    $lines = curl.exe -s -o NUL -D - --max-redirs 0 $Url
    $statusLine = $lines | Select-Object -First 1
    $locationLine = $lines | Where-Object { $_ -match '^Location:' } | Select-Object -First 1
    $status = [int](($statusLine -split ' ')[1])
    $location = ""
    if ($locationLine) {
        $location = ($locationLine -replace '^Location:\s*', '').Trim()
    }
    return [pscustomobject]@{
        Status = $status
        Location = $location
        Raw = ($lines -join " | ")
    }
}

Write-Host "Checking readiness..."
$ready = Invoke-RestMethod -Uri "$BaseUrl/readyz"
Assert-True ($ready.status -eq "ready") "service should be ready"

Write-Host "Creating QR code..."
$createBody = @{ targetUrl = "https://example.com/smoke-start" } | ConvertTo-Json
$created = Invoke-RestMethod `
    -Method Post `
    -Uri "$BaseUrl/api/qr/create" `
    -Headers @{ Authorization = "Bearer $ApiKey" } `
    -ContentType "application/json" `
    -Body $createBody

$token = $created.token
Assert-True ($token.Length -eq 12) "token should be 12 chars"
Write-Host "Created token: $token"

Write-Host "Checking image endpoint..."
$imageHeaders = curl.exe -s -o NUL -D - "$BaseUrl/api/qr/$token/image"
$imageHeaderText = $imageHeaders -join "`n"
Assert-True ($imageHeaderText -match "HTTP/1.1 200 OK") "image endpoint should return 200"
Assert-True ($imageHeaderText -match "Content-Type: image/png") "image endpoint should return image/png"
Assert-True ($imageHeaderText -match "Cache-Control: public, max-age=31536000, immutable") "image endpoint should be cacheable"

Write-Host "Checking redirect and active cache TTL..."
$redirect1 = Get-RedirectHeaders "$BaseUrl/r/$token"
$redirect2 = Get-RedirectHeaders "$BaseUrl/r/$token"
Assert-True ($redirect1.Status -eq 302) "first redirect should return 302"
Assert-True ($redirect2.Status -eq 302) "second redirect should return 302"
Assert-True ($redirect1.Location -eq "https://example.com/smoke-start") "redirect should use initial target"

Start-Sleep -Milliseconds 300
$activeTtl = [int](docker exec qrcode-redis-1 redis-cli TTL "redirect:$token")
Assert-True ($activeTtl -gt 0 -and $activeTtl -le 600) "active cache TTL should be within 10 minutes"

Write-Host "Updating target URL..."
$updateBody = @{ targetUrl = "https://example.org/smoke-updated" } | ConvertTo-Json
$updated = Invoke-RestMethod `
    -Method Patch `
    -Uri "$BaseUrl/api/qr/$token" `
    -Headers @{ Authorization = "Bearer $ApiKey" } `
    -ContentType "application/json" `
    -Body $updateBody
Assert-True ($updated.targetUrl -eq "https://example.org/smoke-updated") "update should return new target"

$redirectAfterUpdate = Get-RedirectHeaders "$BaseUrl/r/$token"
Assert-True ($redirectAfterUpdate.Status -eq 302) "redirect after update should return 302"
Assert-True ($redirectAfterUpdate.Location -eq "https://example.org/smoke-updated") "redirect should use updated target"

Write-Host "Deleting QR code and checking tombstone cache..."
$deleted = Invoke-RestMethod `
    -Method Delete `
    -Uri "$BaseUrl/api/qr/$token" `
    -Headers @{ Authorization = "Bearer $ApiKey" }
Assert-True ($deleted.status -eq "deleted") "delete should return deleted"

$redirectAfterDelete = Get-RedirectHeaders "$BaseUrl/r/$token"
Assert-True ($redirectAfterDelete.Status -eq 410) "redirect after delete should return 410"

Start-Sleep -Milliseconds 300
$tombstoneTtl = [int](docker exec qrcode-redis-1 redis-cli TTL "redirect:$token")
Assert-True ($tombstoneTtl -gt 0 -and $tombstoneTtl -le 300) "tombstone cache TTL should be within 5 minutes"

Write-Host "Checking rate limit headers..."
$metadataHeaders = curl.exe -s -o NUL -D - -H "Authorization: Bearer $ApiKey" "$BaseUrl/api/qr/$token"
$metadataHeaderText = $metadataHeaders -join "`n"
Assert-True ($metadataHeaderText -match "RateLimit-Limit:") "owner API should include RateLimit-Limit"
Assert-True ($metadataHeaderText -match "RateLimit-Remaining:") "owner API should include RateLimit-Remaining"

Write-Host "Smoke test passed."
Write-Host "Token: $token"
Write-Host "Active TTL observed: $activeTtl seconds"
Write-Host "Tombstone TTL observed: $tombstoneTtl seconds"
