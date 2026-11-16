# AUTH-27: end-to-end password reset, email code lifecycle and session revocation.
# Run from repository root with PowerShell 5.1+ and a DEDICATED local Docker test stack.
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

function Assert-Status([string]$Name, [hashtable]$Result, [int]$Expected) {
    if ($Result.Status -ne $Expected) {
        throw ('{0}: expected HTTP {1}, got {2}; response: {3}' -f `
            $Name, $Expected, $Result.Status, $Result.Body)
    }
    $script:passed++
    Write-Host "[PASS] $Name"
}

function Invoke-AuthPost([string]$Path, [hashtable]$Data,
                         [Microsoft.PowerShell.Commands.WebRequestSession]$Session = $null) {
    $body = ConvertTo-Json -InputObject $Data -Compress
    $params = @{
        Uri = "$BaseUrl$Path"; Method = 'Post'; ContentType = 'application/json'
        Body = $body; UseBasicParsing = $true; TimeoutSec = 30
    }
    if ($null -ne $Session) { $params.WebSession = $Session }
    try {
        $response = Invoke-WebRequest @params
        return @{ Status = [int]$response.StatusCode; Body = [string]$response.Content; Headers = $response.Headers }
    }
    catch [System.Net.WebException] {
        $response = $_.Exception.Response
        if ($null -eq $response) { throw }
        try {
            $reader = New-Object System.IO.StreamReader($response.GetResponseStream())
            try { $content = $reader.ReadToEnd() } finally { $reader.Dispose() }
            return @{ Status = [int]$response.StatusCode; Body = $content; Headers = $response.Headers }
        }
        finally { $response.Dispose() }
    }
}

function Invoke-TestSql([string]$Sql) {
    $output = $Sql | & docker compose -f compose.yaml exec -T postgres sh -c `
        'exec psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -v ON_ERROR_STOP=1 -At'
    if ($LASTEXITCODE -ne 0) { throw 'Test PostgreSQL query failed' }
    return (($output | Out-String).Trim())
}

function Get-CookieValue([Microsoft.PowerShell.Commands.WebRequestSession]$Session,
                         [string]$Name) {
    $cookie = $Session.Cookies.GetCookies([uri]$BaseUrl)[$Name]
    if ($null -eq $cookie) { return '' }
    return [string]$cookie.Value
}

function Get-MailMessages([string]$Address, [string]$Purpose) {
    $query = [uri]::EscapeDataString("to:$Address")
    $result = Invoke-RestMethod -Uri "$MailpitUrl/api/v1/search?query=$query" -TimeoutSec 10
    return @($result.messages | Where-Object {
        $message = $_
        $recipients = @($message.To | Where-Object { $_.Address -eq $Address })
        $recipients.Count -gt 0 -and ([string]$message.Subject).ToLowerInvariant().Contains($Purpose)
    })
}

function Wait-EmailCode([string]$Address, [string]$Purpose, [int]$PreviousCount) {
    for ($i = 0; $i -lt 40; $i++) {
        $messages = @(Get-MailMessages $Address $Purpose)
        if ($messages.Count -gt $PreviousCount) {
            # Latest message is the newest message for the unique test recipient.
            $message = Invoke-RestMethod -Uri "$MailpitUrl/api/v1/message/$($messages[0].ID)" -TimeoutSec 10
            $match = [regex]::Match([string]$message.Text, '(?i)code is\s+(\d{6})')
            if (-not $match.Success) { throw 'Verification email format was unexpected' }
            return $match.Groups[1].Value
        }
        Start-Sleep -Milliseconds 300
    }
    throw "No new '$Purpose' email received for test account"
}

function Get-Introspection([string]$Token) {
    $result = Invoke-AuthPost '/internal/introspect' @{ token = $Token }
    Assert-Status 'Introspection returns HTTP 200' $result 200
    return (ConvertFrom-Json -InputObject $result.Body)
}

try {
    $health = Invoke-RestMethod -Uri "$BaseUrl/health" -TimeoutSec 10
    Assert-True 'AUTH is healthy' ($health.status -eq 'ok')
    $null = Invoke-RestMethod -Uri "$MailpitUrl/api/v1/info" -TimeoutSec 10
    Assert-True 'Mailpit is reachable' $true
    Assert-True 'PostgreSQL is reachable' ((Invoke-TestSql 'SELECT 1;') -eq '1')

    $suffix = [guid]::NewGuid().ToString('N').Substring(0, 12)
    $script:login = "auth27_$suffix"
    $email = "$($script:login)@example.test"
    $oldPassword = 'OriginalPassword123!'
    $newPassword = 'ChangedPassword456!'

    $signup = Invoke-AuthPost '/auth/signup' @{
        login = $script:login; email = $email; password = $oldPassword
    }
    Assert-Status 'Test user registered' $signup 201
    $script:created = $true

    # Confirm that psql and HTTP AUTH use the same database before any test-only SQL writes.
    $dbUserCount = Invoke-TestSql "SELECT count(*) FROM users WHERE login = '$($script:login)';"
    if ($dbUserCount -ne '1') {
        throw ("HTTP signup succeeded but Docker PostgreSQL cannot see test user '$($script:login)' (count=$dbUserCount). " +
               'Check that AUTH and the psql test command use the same PostgreSQL database/container.')
    }
    Assert-True 'Test user visible in Docker PostgreSQL' $true

    $recoveryCodes = ConvertFrom-Json -InputObject $signup.Body
    Assert-True 'Recovery code issued' ($recoveryCodes.Count -eq 10 -and $recoveryCodes[0] -is [string])

    # Establish an actual full session BEFORE password reset, using all three factors.
    $initialSession = New-Object Microsoft.PowerShell.Commands.WebRequestSession
    $beforeVerification = @(Get-MailMessages $email 'verification').Count
    $loginResponse = Invoke-AuthPost '/auth/login' @{
        login = $script:login; password = $oldPassword
    } $initialSession
    Assert-Status 'Initial password accepted' $loginResponse 204
    Assert-True 'Initial auth_pending cookie received' `
        (-not [string]::IsNullOrEmpty((Get-CookieValue $initialSession 'auth_pending')))
    $loginCode = Wait-EmailCode $email 'verification' $beforeVerification
    $factor2 = Invoke-AuthPost '/auth/login/email-code' @{ code = $loginCode } $initialSession
    Assert-Status 'Email login factor accepted' $factor2 204
    $factor3 = Invoke-AuthPost '/auth/login/recovery-code' @{
        code = [string]$recoveryCodes[0]
    } $initialSession
    Assert-Status 'Recovery login factor accepted' $factor3 204
    $oldToken = Get-CookieValue $initialSession 'session'
    Assert-True 'Full session issued' (-not [string]::IsNullOrEmpty($oldToken))
    $activeBefore = Get-Introspection $oldToken
    Assert-True 'Old session initially active' ($activeBefore.active -eq $true)

    # First password reset request: test invalid and expired code.
    $expiredSession = New-Object Microsoft.PowerShell.Commands.WebRequestSession
    $beforeReset = @(Get-MailMessages $email 'password reset').Count
    $restore = Invoke-AuthPost '/auth/password/restore' @{ email = $email } $expiredSession
    Assert-Status 'Password reset request accepted' $restore 204
    Assert-True 'Reset pending cookie issued' `
        (-not [string]::IsNullOrEmpty((Get-CookieValue $expiredSession 'auth_reset_pending')))
    $expiredCode = Wait-EmailCode $email 'password reset' $beforeReset
    Assert-True 'Six-digit reset email code received' ($expiredCode -match '^\d{6}$')

    $wrongCode = if ($expiredCode -ne '000000') { '000000' } else { '111111' }
    $invalid = Invoke-AuthPost '/auth/password/confirm' @{ code = $wrongCode } $expiredSession
    Assert-Status 'Incorrect reset code rejected' $invalid 401

    $resetRowCount = Invoke-TestSql "SELECT count(*) FROM password_reset_challenges WHERE user_id = (SELECT id FROM users WHERE login = '$($script:login)');"
    if ($resetRowCount -ne '1') {
        throw "Expected one password reset challenge in Docker PostgreSQL, found $resetRowCount. Check database connections."
    }
    Assert-True 'Password reset challenge visible in PostgreSQL' $true

    $updated = Invoke-TestSql @"
UPDATE password_reset_challenges SET expires_at = now() - interval '1 second'
WHERE user_id = (SELECT id FROM users WHERE login = '$($script:login)');
"@
    if ($updated -ne 'UPDATE 1') {
        throw "Could not expire this test user's reset challenge (SQL returned '$updated')."
    }
    Assert-True 'Test reset challenge expired (one row)' $true
    $expired = Invoke-AuthPost '/auth/password/confirm' @{ code = $expiredCode } $expiredSession
    Assert-Status 'Expired reset code rejected' $expired 401

    # Remove only this test user's expired challenge to start a fresh reset.
    $deleted = Invoke-TestSql @"
DELETE FROM password_reset_challenges
WHERE user_id = (SELECT id FROM users WHERE login = '$($script:login)');
"@
    Assert-True 'Expired test challenge cleared' ($deleted -eq 'DELETE 1')

    $resetSession = New-Object Microsoft.PowerShell.Commands.WebRequestSession
    $beforeReset = @(Get-MailMessages $email 'password reset').Count
    $restore = Invoke-AuthPost '/auth/password/restore' @{ email = $email } $resetSession
    Assert-Status 'Fresh password reset request accepted' $restore 204
    $pending = Get-CookieValue $resetSession 'auth_reset_pending'
    Assert-True 'Fresh reset cookie issued' (-not [string]::IsNullOrEmpty($pending))
    $resetCode = Wait-EmailCode $email 'password reset' $beforeReset

    $confirmed = Invoke-AuthPost '/auth/password/confirm' @{ code = $resetCode } $resetSession
    Assert-Status 'Correct reset code accepted' $confirmed 204
    Assert-True 'Password-reset grant cookie issued' `
        (-not [string]::IsNullOrEmpty((Get-CookieValue $resetSession 'auth_reset_grant')))

    # Reuse the consumed email code with the SAVED original pending cookie.
    $replaySession = New-Object Microsoft.PowerShell.Commands.WebRequestSession
    $uri = [uri]$BaseUrl
    $replaySession.Cookies.Add((New-Object System.Net.Cookie('auth_reset_pending', $pending, '/', $uri.Host)))
    $reused = Invoke-AuthPost '/auth/password/confirm' @{ code = $resetCode } $replaySession
    Assert-Status 'Used reset code rejected' $reused 401

    $changed = Invoke-AuthPost '/auth/password/confirm' @{ password = $newPassword } $resetSession
    Assert-Status 'New password saved' $changed 204

    $replayPassword = Invoke-AuthPost '/auth/password/confirm' @{ password = $newPassword } $resetSession
    Assert-Status 'Password-reset grant is single-use' $replayPassword 401

    $oldSessionState = Get-Introspection $oldToken
    Assert-True 'Previous full session revoked' ($oldSessionState.active -eq $false)
    $oldLogin = Invoke-AuthPost '/auth/login' @{
        login = $script:login; password = $oldPassword
    }
    Assert-Status 'Old password rejected' $oldLogin 401

    $newLogin = Invoke-AuthPost '/auth/login' @{
        login = $script:login; password = $newPassword
    }
    Assert-Status 'New password accepted' $newLogin 204
    Assert-True 'New login issues pending cookie' `
        ([string]$newLogin.Headers['Set-Cookie'] -match 'auth_pending=')

    Write-Host "`nAUTH-27 E2E PASSED: $script:passed checks"
}
catch {
    Write-Error "AUTH-27 E2E FAILED: $_"
    exit 1
}
finally {
    if ($script:created -and $script:login) {
        try {
            $deleted = Invoke-TestSql "DELETE FROM users WHERE login = '$($script:login)';"
            if ($deleted -eq 'DELETE 1') { Write-Host '[CLEANUP] Test user removed' }
            else { Write-Warning "Unexpected cleanup result: $deleted. Check for a mismatched AUTH/PostgreSQL database connection." }
        }
        catch { Write-Warning "Failed to remove test user: $_" }
    }
}
