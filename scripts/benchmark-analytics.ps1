param(
    [string]$BaseUrl = "http://localhost:8080",
    [string]$K6BaseUrl = "http://host.docker.internal:8080",
    [string]$ApiKey = "qk_demo_local_dev_key",
    [int]$Rate = 500,
    [string]$Duration = "30s",
    [int]$VUs = 100,
    [int]$MaxVUs = 500,
    [int]$DrainTimeoutSeconds = 60
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

    $createBody = @{ targetUrl = "https://example.com/benchmark-analytics" } | ConvertTo-Json
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

function Get-MetricSnapshot {
    param(
        [string]$Url,
        [string[]]$Series
    )

    $metricsText = Get-MetricsText -Url $Url
    $snapshot = @{}
    foreach ($item in $Series) {
        $snapshot[$item] = Get-MetricValue -MetricsText $metricsText -Series $item
    }
    return $snapshot
}

function Show-MetricDeltas {
    param(
        [hashtable]$Before,
        [hashtable]$After,
        [string[]]$Series
    )

    $rows = foreach ($item in $Series) {
        [pscustomobject]@{
            Metric = $item
            Before = $Before[$item]
            After = $After[$item]
            Delta = $After[$item] - $Before[$item]
        }
    }
    $rows | Format-Table -AutoSize
}

function Wait-AnalyticsDrain {
    param(
        [string]$Url,
        [double]$BaselineWritten,
        [int]$ExpectedNewEvents,
        [int]$TimeoutSeconds
    )

    $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
    $last = $null

    while ((Get-Date) -lt $deadline) {
        $metricsText = Get-MetricsText -Url $Url
        $written = Get-MetricValue -MetricsText $metricsText -Series "analytics_events_written_total"
        $pending = Get-MetricValue -MetricsText $metricsText -Series "analytics_events_pending"
        $failures = Get-MetricValue -MetricsText $metricsText -Series "analytics_worker_failures_total"
        $last = [pscustomobject]@{
            WrittenDelta = $written - $BaselineWritten
            Pending = $pending
            Failures = $failures
        }

        if ($last.WrittenDelta -ge $ExpectedNewEvents -and $last.Pending -eq 0) {
            return $last
        }

        Start-Sleep -Seconds 1
    }

    throw "analytics did not drain within $TimeoutSeconds seconds; last written delta=$($last.WrittenDelta), pending=$($last.Pending), failures=$($last.Failures)"
}

$series = @(
    "redirect_requests_total{result=""redirect""}",
    "redirect_cache_hits_total",
    "redirect_db_lookups_total",
    "analytics_enqueue_failures_total",
    "analytics_events_written_total",
    "analytics_batches_written_total",
    "analytics_events_reclaimed_total",
    "analytics_worker_failures_total",
    "analytics_events_pending",
    "analytics_stream_length",
    "analytics_batch_write_duration_seconds_count",
    "redirect_latency_seconds_count"
)

Write-Host "Starting API with analytics worker enabled..."
$env:REDIRECT_CACHE_ENABLED = "true"
$env:ANALYTICS_WORKER_ENABLED = "true"
docker compose up -d --build api
Wait-Ready -Url $BaseUrl

$created = New-TestQRCode -Url $BaseUrl -Key $ApiKey
$token = $created.token
Write-Host "Benchmark token: $token"

Write-Host "Warming redirect cache..."
curl.exe -s -o NUL --max-redirs 0 "$BaseUrl/r/$token"
Start-Sleep -Seconds 2

$before = Get-MetricSnapshot -Url $BaseUrl -Series $series

Write-Host "Running k6 redirect load at $Rate RPS for $Duration..."
docker run --rm `
    -v "${PWD}/load-tests:/scripts" `
    -e BASE_URL=$K6BaseUrl `
    -e TOKEN=$token `
    -e RATE=$Rate `
    -e DURATION=$Duration `
    -e VUS=$VUs `
    -e MAX_VUS=$MaxVUs `
    grafana/k6 run /scripts/redirect.k6.js

$requests = [int](Get-MetricValue -MetricsText (Get-MetricsText -Url $BaseUrl) -Series "redirect_latency_seconds_count")
$expectedEvents = $requests - [int]$before["redirect_latency_seconds_count"]

Write-Host "Waiting for analytics worker to drain $expectedEvents new events..."
$drain = Wait-AnalyticsDrain `
    -Url $BaseUrl `
    -BaselineWritten $before["analytics_events_written_total"] `
    -ExpectedNewEvents $expectedEvents `
    -TimeoutSeconds $DrainTimeoutSeconds

$after = Get-MetricSnapshot -Url $BaseUrl -Series $series

Write-Host ""
Write-Host "Drain result:"
$drain | Format-List

Write-Host ""
Write-Host "Metric deltas:"
Show-MetricDeltas -Before $before -After $after -Series $series
