
$ErrorActionPreference = "Stop"

try {
    $response = Invoke-RestMethod `
        -Uri "http://127.0.0.1:8080/health" `
        -Method Get `
        -TimeoutSec 5

    if ($response.status -ne "ok") {
        throw "Unexpected health status"
    }

    if ($response.service -ne "auth") {
        throw "Unexpected service name"
    }

    Write-Host "AUTH health check: PASSED"
    exit 0
}
catch {
    Write-Error "AUTH health check: FAILED - $_"
    exit 1
}
