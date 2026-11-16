# AUTH-26: end-to-end checks for login, SMTP delivery, and account lockout.
# Run from the repository root. Requires PowerShell 5.1+, Docker Compose,
# running AUTH/PostgreSQL/Mailpit and a dedicated local test database.
param(
    [string]$BaseUrl = 'http://127.0.0.1:8080',
    [string]$MailpitUrl = 'http://127.0.0.1:8025'
)

$ErrorActionPreference = 'Stop'
$BaseUrl = $BaseUrl.TrimEnd('/')
$MailpitUrl = $MailpitUrl.TrimEnd('/')
$script:passed = 0
$script:created = $false
$script:login = $null

function Assert-True([string]$Name, [bool]$Condition) {
    if (-not $Condition) { throw "FAIL: $Name" }
    $script:passed++
    Write-Host "[PASS] $Name"
}

function Invoke-AuthPost([string]$Path, [hashtable]$Data) {
    $body = ConvertTo-Json -InputObject $Data -Compress
    try {
        $response = Invoke-WebRequest -Uri "$BaseUrl$Path" -Method Post `
            -ContentType 'application/json' -Body $body -UseBasicParsing -TimeoutSec 30
        return @{ Status = [int]$response.StatusCode; Headers = $response.Headers; Body = $response.Content }
    }
    catch [System.Net.WebException] {
        if ($null -eq $_.Exception.Response) { throw }
        $response = $_.Exception.Response
        try {
            return @{ Status = [int]$response.StatusCode; Headers = $response.Headers; Body = '' }
        }
        finally { $response.Dispose() }
    }
}

function Assert-Status([string]$Name, [hashtable]$Response, [int]$Expected) {
    Assert-True $Name ($Response.Status -eq $Expected)
    if ($Response.Status -ne $Expected) {
        throw "${Name}: expected $Expected but got $($Response.Status)"
    }
}

# Local-only database introspection/clock adjustment. No waiting for 15 minutes.
function Invoke-TestSql([string]$Sql) {
    $output = $Sql | & docker compose -f compose.yaml exec -T postgres sh -c `
        'exec psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -v ON_ERROR_STOP=1 -At'
    if ($LASTEXITCODE -ne 0) { throw 'PostgreSQL test command failed' }
    return (($output | Out-String).Trim())
}

function Get-MailCount([string]$Address) {
    $query = [uri]::EscapeDataString("to:$Address")
    $result = Invoke-RestMethod -Uri "$MailpitUrl/api/v1/search?query=$query" -TimeoutSec 10
    # Verify exact recipient, not a fuzzy search match.
    $matched = @($result.messages | Where-Object {
        @($_.To | Where-Object { $_.Address -eq $Address }).Count -gt 0
    })
    return $matched.Count
}

try {
    # Fail early when prerequisites are unavailable.
    $health = Invoke-RestMethod -Uri "$BaseUrl/health" -TimeoutSec 10
    Assert-True 'AUTH health is OK' ($health.status -eq 'ok')
    $null = Invoke-RestMethod -Uri "$MailpitUrl/api/v1/info" -TimeoutSec 10
    Assert-True 'Mailpit API is available' $true
    $null = Invoke-TestSql 'SELECT 1;'
    Assert-True 'Test PostgreSQL is available' $true

    $suffix = [guid]::NewGuid().ToString('N').Substring(0, 12)
    $script:login = "auth26_$suffix"
    $email = "$($script:login)@example.test"
    $password = 'TestPassword123!'
    $badPassword = 'WrongPassword123!'

    $signup = Invoke-AuthPost '/auth/signup' @{
        login = $script:login; email = $email; password = $password
    }
    Assert-Status 'Test account registration' $signup 201
    $script:created = $true

    $notFound = Invoke-AuthPost '/auth/login' @{
        login = "unknown_$suffix"; password = $password
    }
    Assert-Status 'Unknown login is rejected' $notFound 401

    $first = Invoke-AuthPost '/auth/login' @{
        login = $script:login; password = $password
    }
    Assert-Status 'Correct password returns HTTP 204' $first 204
    Assert-True 'auth_pending cookie issued' ([string]$first.Headers['Set-Cookie'] -match 'auth_pending=')

    $mailBefore = 0
    for ($i = 0; $i -lt 10; $i++) {
        $mailBefore = Get-MailCount $email
        if ($mailBefore -ge 1) { break }
        Start-Sleep -Milliseconds 500
    }
    Assert-True 'Email delivery visible in Mailpit' ($mailBefore -ge 1)

    for ($attempt = 1; $attempt -le 5; $attempt++) {
        $result = Invoke-AuthPost '/auth/login' @{
            login = $script:login; password = $badPassword
        }
        Assert-Status "Bad password attempt $attempt returns 401" $result 401
        $state = Invoke-TestSql "SELECT failed_login_attempts FROM users WHERE login = '$($script:login)';"
        Assert-True "Failure counter is $attempt" ($state -eq [string]$attempt)
    }

    $lockState = Invoke-TestSql @"
SELECT (login_locked_until > now())::int || '|' ||
       (EXTRACT(EPOCH FROM (login_locked_until - now()))::int)::text
FROM users WHERE login = '$($script:login)';
"@
    $parts = $lockState -split '\|'
    Assert-True 'Account is locked after 5 failed attempts' ($parts.Count -eq 2 -and $parts[0] -eq '1')
    $secondsLeft = [int]$parts[1]
    Assert-True 'Lock duration is approximately 15 minutes' ($secondsLeft -ge 840 -and $secondsLeft -le 910)

    $locked = Invoke-AuthPost '/auth/login' @{
        login = $script:login; password = $password
    }
    Assert-Status 'Correct password rejected while locked' $locked 401
    Assert-True 'Locked request does not send email' ((Get-MailCount $email) -eq $mailBefore)

    # ONLY the just-created test user. Simulates the passing of 15 minutes.
    $updated = Invoke-TestSql @"
UPDATE users SET login_locked_until = now() - interval '1 second'
WHERE login = '$($script:login)' AND login_locked_until > now();
"@
    Assert-True 'Test clock adjusted (one row)' ($updated -eq 'UPDATE 1')

    $unlocked = Invoke-AuthPost '/auth/login' @{
        login = $script:login; password = $password
    }
    Assert-Status 'Correct password accepted after lock expiration' $unlocked 204
    Assert-True 'New auth_pending cookie issued' ([string]$unlocked.Headers['Set-Cookie'] -match 'auth_pending=')

    $countAfter = 0
    for ($i = 0; $i -lt 10; $i++) {
        $countAfter = Get-MailCount $email
        if ($countAfter -ge ($mailBefore + 1)) { break }
        Start-Sleep -Milliseconds 500
    }
    Assert-True 'Second successful login sends another email' ($countAfter -ge ($mailBefore + 1))

    $reset = Invoke-TestSql "SELECT failed_login_attempts FROM users WHERE login = '$($script:login)';"
    Assert-True 'Successful login resets failed-attempt counter' ($reset -eq '0')

    Write-Host "`nAUTH-26 E2E PASSED: $script:passed checks"
}
catch {
    Write-Error "AUTH-26 E2E FAILED: $_"
    exit 1
}
finally {
    if ($script:created -and $script:login) {
        try {
            # This is a test-only account. Dependent rows are removed by FK cascade.
            $null = Invoke-TestSql "DELETE FROM users WHERE login = '$($script:login)';"
            Write-Host '[CLEANUP] Test user removed'
        }
        catch { Write-Warning "Failed to remove test user: $_" }
    }
}
