# AUTH-30: check simultaneous AUTH/COORD deployment and signup proxy.
# Run from AUTH repository root, PowerShell 5.1+, on a dedicated LOCAL test database.
param(
    [string]$AuthUrl = 'http://127.0.0.1:8081',
    [string]$CoordUrl = 'http://127.0.0.1:8080'
)
$ErrorActionPreference = 'Stop'
$AuthUrl = $AuthUrl.TrimEnd('/')
$CoordUrl = $CoordUrl.TrimEnd('/')
$script:passed = 0
$script:created = $false
$script:testLogin = ''

function Assert-True([string]$Name, [bool]$Condition) {
    if (-not $Condition) { throw ('FAIL: {0}' -f $Name) }
    $script:passed++
    Write-Host ('[PASS] {0}' -f $Name)
}
function Invoke-TestSql([string]$Sql) {
    $output = $Sql | & docker compose -f compose.yaml exec -T postgres sh -c `
        'exec psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -v ON_ERROR_STOP=1 -At'
    if ($LASTEXITCODE -ne 0) { throw 'AUTH PostgreSQL SQL command failed' }
    return (($output | Out-String).Trim())
}
function Invoke-Post([string]$Url, [hashtable]$Data) {
    try {
        $r = Invoke-WebRequest -Uri $Url -Method Post -ContentType 'application/json' `
            -Body (ConvertTo-Json -InputObject $Data -Compress) -UseBasicParsing -TimeoutSec 30
        return @{ Status=[int]$r.StatusCode; Body=[string]$r.Content }
    }
    catch [System.Net.WebException] {
        if ($null -eq $_.Exception.Response) { throw }
        $r = $_.Exception.Response
        try {
            $reader = New-Object System.IO.StreamReader($r.GetResponseStream())
            try { $content=$reader.ReadToEnd() } finally { $reader.Dispose() }
            return @{ Status=[int]$r.StatusCode; Body=$content }
        }
        finally { $r.Dispose() }
    }
}
try {
    $authHealth = Invoke-RestMethod -Uri "$AuthUrl/health" -TimeoutSec 10
    Assert-True 'AUTH /health reachable on host port 8081' ($authHealth.status -eq 'ok')

    $coordHealth = Invoke-RestMethod -Uri "$CoordUrl/health" -TimeoutSec 10
    Assert-True 'COORD /health reachable on host port 8080' ($coordHealth.status -eq 'ok')

    Assert-True 'AUTH PostgreSQL reachable' ((Invoke-TestSql 'SELECT 1;') -eq '1')

    $suffix = [guid]::NewGuid().ToString('N').Substring(0,12)
    $script:testLogin = "auth30_$suffix"
    $email = "$($script:testLogin)@example.test"
    $signup = Invoke-Post "$CoordUrl/api/auth/signup" @{
        login = $script:testLogin
        email = $email
        password = 'TestPassword123!'
    }
    if ($signup.Status -ne 201) {
        throw ('COORD signup expected HTTP 201; got {0}; body: {1}' -f $signup.Status, $signup.Body)
    }
    $script:created = $true
    Assert-True 'COORD proxies /api/auth/signup to AUTH (201)' $true

    $codes = ConvertFrom-Json -InputObject $signup.Body
    Assert-True 'COORD preserves ten recovery codes' ($codes.Count -eq 10)

    $count = Invoke-TestSql "SELECT count(*) FROM users WHERE login = '$($script:testLogin)';"
    Assert-True 'COORD signup persisted in AUTH PostgreSQL' ($count -eq '1')

    $duplicate = Invoke-Post "$CoordUrl/api/auth/signup" @{
        login = $script:testLogin
        email = $email
        password = 'TestPassword123!'
    }
    Assert-True 'COORD forwards duplicate-account conflict (409)' ($duplicate.Status -eq 409)

    Write-Host ("`nAUTH-30 INTEGRATION PASSED: {0} checks" -f $script:passed)
}
catch {
    Write-Error ('AUTH-30 INTEGRATION FAILED: {0}' -f $_)
    exit 1
}
finally {
    if ($script:created -and $script:testLogin) {
        try {
            $deleted = Invoke-TestSql "DELETE FROM users WHERE login = '$($script:testLogin)';"
            if ($deleted -eq 'DELETE 1') { Write-Host '[CLEANUP] Test user removed' }
            else { Write-Warning ('Unexpected cleanup result: {0}' -f $deleted) }
        }
        catch { Write-Warning ('Cleanup failed: {0}' -f $_) }
    }
}
