param(
    [ValidateSet("true", "false")]
    [string]$CacheEnabled = "true",
    [string]$BaseUrl = "http://localhost:8080",
    [string]$K6BaseUrl = "http://host.docker.internal:8080",
    [string]$ApiKey = "qk_demo_local_dev_key",
    [int]$Rate = 100,
    [string]$Duration = "30s",
    [int]$VUs = 50,
    [int]$MaxVUs = 200
)

$ErrorActionPreference = "Stop"

function Wait-Ready {
    param([string]$Url)

    for ($i = 0; $i -lt 30; $i++) {
        try {
            $ready = Invoke-RestMethod -Uri "$Url/readyz" -TimeoutSec 2
            if ($ready.status -eq "ready") {
                return
            }
        } catch {
            Start-Sleep -Seconds 1
        }
    }

    throw "service did not become ready at $Url"
}

function New-TestQRCode {
    param(
        [string]$Url,
        [string]$Key
    )

    $createBody = @{ targetUrl = "https://example.com/benchmark-redirect" } | ConvertTo-Json
    return Invoke-RestMethod `
        -Method Post `
        -Uri "$Url/api/qr/create" `
        -Headers @{ Authorization = "Bearer $Key" } `
        -ContentType "application/json" `
        -Body $createBody
}

function Get-MetricsText {
    param([string]$Url)
    return (Invoke-WebRequest -Uri "$Url/metrics").Content
}

function Get-MetricValue {
    param(
        [string]$MetricsText,
        [string]$Series
    )

    $line = $MetricsText -split "`n" |
        Where-Object { $_.StartsWith("$Series ") } |
        Select-Object -Last 1

    if (-not $line) {
        return 0.0
    }

    $parts = $line.Trim() -split "\s+"
    return [double]$parts[-1]
}

$previousCacheEnv = $env:REDIRECT_CACHE_ENABLED
$env:REDIRECT_CACHE_ENABLED = $CacheEnabled

try {
    Write-Host "Starting API with REDIRECT_CACHE_ENABLED=$CacheEnabled..."
    docker compose up -d --build api
    Wait-Ready -Url $BaseUrl

    $created = New-TestQRCode -Url $BaseUrl -Key $ApiKey
    $token = $created.token
    Write-Host "Benchmark token: $token"

    if ($CacheEnabled -eq "true") {
        Write-Host "Warming redirect cache..."
        curl.exe -s -o NUL --max-redirs 0 "$BaseUrl/r/$token"
        Start-Sleep -Milliseconds 300
    }

    $series = @(
        "redirect_requests_total{result=""redirect""}",
        "redirect_requests_total{result=""not_found""}",
        "redirect_requests_total{result=""gone""}",
        "redirect_requests_total{result=""error""}",
        "redirect_cache_hits_total",
        "redirect_cache_misses_total",
        "redirect_db_lookups_total",
        "analytics_enqueue_failures_total",
        "redirect_latency_seconds_count"
    )

    $beforeText = Get-MetricsText -Url $BaseUrl
    $before = @{}
    foreach ($item in $series) {
        $before[$item] = Get-MetricValue -MetricsText $beforeText -Series $item
    }

    Write-Host "Running k6 at $Rate RPS for $Duration..."
    docker run --rm `
        -v "${PWD}/load-tests:/scripts" `
        -e BASE_URL=$K6BaseUrl `
        -e TOKEN=$token `
        -e RATE=$Rate `
        -e DURATION=$Duration `
        -e VUS=$VUs `
        -e MAX_VUS=$MaxVUs `
        grafana/k6 run /scripts/redirect.k6.js

    $afterText = Get-MetricsText -Url $BaseUrl
    $rows = foreach ($item in $series) {
        $after = Get-MetricValue -MetricsText $afterText -Series $item
        [pscustomobject]@{
            Metric = $item
            Before = $before[$item]
            After = $after
            Delta = $after - $before[$item]
        }
    }

    Write-Host ""
    Write-Host "Metric deltas:"
    $rows | Format-Table -AutoSize
} finally {
    $env:REDIRECT_CACHE_ENABLED = $previousCacheEnv
}
