# AUTH-28: End-to-end tests for three-factor authentication.
# PowerShell 5.1+, Docker Compose, AUTH + PostgreSQL + Mailpit.
# Run ONLY against a dedicated local test database.
param(
    [string]$BaseUrl = 'http://127.0.0.1:8080',
    [string]$MailpitUrl = 'http://127.0.0.1:8025'
)

$ErrorActionPreference = 'Stop'
$BaseUrl = $BaseUrl.TrimEnd('/')
$MailpitUrl = $MailpitUrl.TrimEnd('/')
$script:passed = 0
$script:createdLogins = @()

function Assert-True([string]$Name, [bool]$Condition) {
    if (-not $Condition) { throw ('FAIL: {0}' -f $Name) }
    $script:passed++
    Write-Host ('[PASS] {0}' -f $Name)
}

function Assert-Status([string]$Name, [hashtable]$Result, [int]$Expected) {
    if ($Result.Status -ne $Expected) {
        throw ('{0}: expected HTTP {1}, got {2}; body={3}' -f $Name, $Expected, $Result.Status, $Result.Body)
    }
    $script:passed++
    Write-Host ('[PASS] {0}' -f $Name)
}

function Invoke-AuthPost(
    [string]$Path,
    [hashtable]$Data,
    [Microsoft.PowerShell.Commands.WebRequestSession]$Session = $null
) {
    $parameters = @{
        Uri = "$BaseUrl$Path"
        Method = 'Post'
        ContentType = 'application/json'
        Body = (ConvertTo-Json -InputObject $Data -Compress)
        UseBasicParsing = $true
        TimeoutSec = 30
    }
    if ($null -ne $Session) { $parameters.WebSession = $Session }
    try {
        $r = Invoke-WebRequest @parameters
        return @{ Status = [int]$r.StatusCode; Body = [string]$r.Content; Headers = $r.Headers }
    }
    catch [System.Net.WebException] {
        $r = $_.Exception.Response
        if ($null -eq $r) { throw }
        try {
            $reader = New-Object System.IO.StreamReader($r.GetResponseStream())
            try { $body = $reader.ReadToEnd() } finally { $reader.Dispose() }
            return @{ Status = [int]$r.StatusCode; Body = $body; Headers = $r.Headers }
        }
        finally { $r.Dispose() }
    }
}

function Get-Cookie(
    [Microsoft.PowerShell.Commands.WebRequestSession]$Session,
    [string]$CookieName
) {
    $cookie = $Session.Cookies.GetCookies([uri]$BaseUrl)[$CookieName]
    if ($null -eq $cookie) { return '' }
    return [string]$cookie.Value
}

function Invoke-TestSql([string]$Sql) {
    $output = $Sql | & docker compose -f compose.yaml exec -T postgres sh -c `
        'exec psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -v ON_ERROR_STOP=1 -At'
    if ($LASTEXITCODE -ne 0) { throw 'Test PostgreSQL query failed' }
    return (($output | Out-String).Trim())
}

function Get-VerificationMessages([string]$Address) {
    $q = [uri]::EscapeDataString("to:$Address")
    $data = Invoke-RestMethod -Uri "$MailpitUrl/api/v1/search?query=$q" -TimeoutSec 10
    return @($data.messages | Where-Object {
        $m = $_
        (@($m.To | Where-Object { $_.Address -eq $Address }).Count -gt 0) -and
        ([string]$m.Subject).ToLowerInvariant().Contains('verification')
    })
}

function Wait-VerificationCode([string]$Address, [int]$PreviousCount) {
    for ($attempt = 0; $attempt -lt 40; $attempt++) {
        $messages = @(Get-VerificationMessages $Address)
        if ($messages.Count -gt $PreviousCount) {
            $newest = $messages | Sort-Object -Property Created -Descending | Select-Object -First 1
            $message = Invoke-RestMethod -Uri "$MailpitUrl/api/v1/message/$($newest.ID)" -TimeoutSec 10
            $match = [regex]::Match([string]$message.Text, '(?i)code is\s+(\d{6})')
            if (-not $match.Success) { throw 'Unexpected verification email format' }
            return $match.Groups[1].Value
        }
        Start-Sleep -Milliseconds 300
    }
    throw ('No verification email for {0}' -f $Address)
}

function Get-Introspection([string]$Token) {
    $r = Invoke-AuthPost '/internal/introspect' @{ token = $Token }
    Assert-Status 'Introspection responds HTTP 200' $r 200
    return (ConvertFrom-Json -InputObject $r.Body)
}

try {
    $health = Invoke-RestMethod -Uri "$BaseUrl/health" -TimeoutSec 10
    Assert-True 'AUTH health is OK' ($health.status -eq 'ok')
    $null = Invoke-RestMethod -Uri "$MailpitUrl/api/v1/info" -TimeoutSec 10
    Assert-True 'Mailpit API is available' $true
    Assert-True 'PostgreSQL is available' ((Invoke-TestSql 'SELECT 1;') -eq '1')

    $suffix = [guid]::NewGuid().ToString('N').Substring(0, 12)
    $userA = "auth28a_$suffix"
    $userB = "auth28b_$suffix"
    $emailA = "$userA@example.test"
    $emailB = "$userB@example.test"
    $password = 'TestPassword123!'

    $signupA = Invoke-AuthPost '/auth/signup' @{ login = $userA; email = $emailA; password = $password }
    Assert-Status 'User A registered' $signupA 201
    $script:createdLogins += $userA
    $codesA = ConvertFrom-Json -InputObject $signupA.Body
    Assert-True 'User A received ten recovery codes' ($codesA.Count -eq 10 -and $codesA[0] -is [string])

    $signupB = Invoke-AuthPost '/auth/signup' @{ login = $userB; email = $emailB; password = $password }
    Assert-Status 'User B registered' $signupB 201
    $script:createdLogins += $userB
    $codesB = ConvertFrom-Json -InputObject $signupB.Body
    Assert-True 'User B received ten recovery codes' ($codesB.Count -eq 10 -and $codesB[0] -is [string])

    $dbA = Invoke-TestSql "SELECT count(*) FROM users WHERE login = '$userA';"
    $dbB = Invoke-TestSql "SELECT count(*) FROM users WHERE login = '$userB';"
    Assert-True 'HTTP users visible in test PostgreSQL' ($dbA -eq '1' -and $dbB -eq '1')

    # Stage 1: correct password. An auth_pending cookie is NOT a full session.
    $sessionA = New-Object Microsoft.PowerShell.Commands.WebRequestSession
    $beforeMail = @(Get-VerificationMessages $emailA).Count
    $firstFactor = Invoke-AuthPost '/auth/login' @{ login = $userA; password = $password } $sessionA
    Assert-Status 'Password factor accepted' $firstFactor 204
    $pendingToken = Get-Cookie $sessionA 'auth_pending'
    Assert-True 'First factor issues auth_pending' ($pendingToken -match '^[0-9a-f]{64}$')
    Assert-True 'First factor does not issue session' ([string]::IsNullOrEmpty((Get-Cookie $sessionA 'session')))
    $beforeEmailState = Get-Introspection $pendingToken
    Assert-True 'auth_pending is not a full session' ($beforeEmailState.active -eq $false)

    # Bypass attempt: cannot use recovery code without email confirmation.
    $skipEmail = Invoke-AuthPost '/auth/login/recovery-code' @{ code = [string]$codesA[0] } $sessionA
    Assert-Status 'Skipping email factor rejected' $skipEmail 401
    Assert-True 'Skipping email factor does not issue session' ([string]::IsNullOrEmpty((Get-Cookie $sessionA 'session')))

    $emailCode = Wait-VerificationCode $emailA $beforeMail
    Assert-True 'Verification code received by SMTP' ($emailCode -match '^\d{6}$')

    # Stage 2: email challenge. It should not yet create a full session.
    $emailFactor = Invoke-AuthPost '/auth/login/email-code' @{ code = $emailCode } $sessionA
    Assert-Status 'Email factor accepted' $emailFactor 204
    $pendingRecovery = Get-Cookie $sessionA 'auth_recovery_pending'
    Assert-True 'Second factor issues auth_recovery_pending' ($pendingRecovery -match '^[0-9a-f]{64}$')
    Assert-True 'Second factor clears auth_pending' ([string]::IsNullOrEmpty((Get-Cookie $sessionA 'auth_pending')))
    Assert-True 'Second factor does not issue session' ([string]::IsNullOrEmpty((Get-Cookie $sessionA 'session')))
    $beforeRecoveryState = Get-Introspection $pendingRecovery
    Assert-True 'auth_recovery_pending is not a full session' ($beforeRecoveryState.active -eq $false)

    # Replaying email code with original first-factor token fails.
    $replayedEmailSession = New-Object Microsoft.PowerShell.Commands.WebRequestSession
    $uri = [uri]$BaseUrl
    $replayedEmailSession.Cookies.Add((New-Object System.Net.Cookie('auth_pending', $pendingToken, '/', $uri.Host)))
    $emailReplay = Invoke-AuthPost '/auth/login/email-code' @{ code = $emailCode } $replayedEmailSession
    Assert-Status 'Consumed email code cannot be reused' $emailReplay 401

    # Stage 3: the code is bound to the owner of the challenge.
    $foreignCode = Invoke-AuthPost '/auth/login/recovery-code' @{ code = [string]$codesB[0] } $sessionA
    Assert-Status 'Other user recovery code rejected' $foreignCode 401
    Assert-True 'Foreign code cannot create session' ([string]::IsNullOrEmpty((Get-Cookie $sessionA 'session')))

    # A rejected third-factor attempt rolls back challenge deletion.
    $recoveryFactor = Invoke-AuthPost '/auth/login/recovery-code' @{ code = [string]$codesA[0] } $sessionA
    Assert-Status 'Own recovery code accepted' $recoveryFactor 204
    $fullToken = Get-Cookie $sessionA 'session'
    Assert-True 'Third factor issues a full session' ($fullToken -match '^[0-9a-f]{64}$')
    Assert-True 'Third factor clears auth_recovery_pending' ([string]::IsNullOrEmpty((Get-Cookie $sessionA 'auth_recovery_pending')))
    $fullState = Get-Introspection $fullToken
    Assert-True 'Full session active after third factor' ($fullState.active -eq $true)
    Assert-True 'Session principal belongs to user A' ($fullState.login -eq $userA -and $fullState.stage -eq 'full')

    # Replaying the original pending challenge cannot issue another session.
    $replayedRecoverySession = New-Object Microsoft.PowerShell.Commands.WebRequestSession
    $replayedRecoverySession.Cookies.Add((New-Object System.Net.Cookie('auth_recovery_pending', $pendingRecovery, '/', $uri.Host)))
    $recoveryReplay = Invoke-AuthPost '/auth/login/recovery-code' @{ code = [string]$codesA[0] } $replayedRecoverySession
    Assert-Status 'Consumed third-factor challenge cannot be reused' $recoveryReplay 401

    # New three-factor attempt, but code A[0] was consumed and must not be reusable.
    $nextSession = New-Object Microsoft.PowerShell.Commands.WebRequestSession
    $beforeMail2 = @(Get-VerificationMessages $emailA).Count
    $newFirstFactor = Invoke-AuthPost '/auth/login' @{ login = $userA; password = $password } $nextSession
    Assert-Status 'Second login password accepted' $newFirstFactor 204
    $nextEmailCode = Wait-VerificationCode $emailA $beforeMail2
    $newEmailFactor = Invoke-AuthPost '/auth/login/email-code' @{ code = $nextEmailCode } $nextSession
    Assert-Status 'Second login email factor accepted' $newEmailFactor 204
    $usedCode = Invoke-AuthPost '/auth/login/recovery-code' @{ code = [string]$codesA[0] } $nextSession
    Assert-Status 'Used recovery code rejected on new challenge' $usedCode 401
    Assert-True 'Used recovery code does not issue session' ([string]::IsNullOrEmpty((Get-Cookie $nextSession 'session')))

    $freshCode = Invoke-AuthPost '/auth/login/recovery-code' @{ code = [string]$codesA[1] } $nextSession
    Assert-Status 'Unused recovery code accepted' $freshCode 204
    Assert-True 'Second login issues a full session' ((Get-Cookie $nextSession 'session') -match '^[0-9a-f]{64}$')

    Write-Host "`nAUTH-28 E2E PASSED: $script:passed checks"
}
catch {
    Write-Error ('AUTH-28 E2E FAILED: {0}' -f $_)
    exit 1
}
finally {
    foreach ($createdUser in $script:createdLogins) {
        try {
            $deleted = Invoke-TestSql "DELETE FROM users WHERE login = '$createdUser';"
            if ($deleted -eq 'DELETE 1') { Write-Host ('[CLEANUP] User {0} removed' -f $createdUser) }
            else { Write-Warning ('Unexpected cleanup result for {0}: {1}' -f $createdUser, $deleted) }
        }
        catch { Write-Warning ('Cleanup failed for {0}: {1}' -f $createdUser, $_) }
    }
}
