# AUTH-29: E2E tests for /users authorization and session revocation.
# Run from repository root, PowerShell 5.1+; requires a DEDICATED LOCAL test database.
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
function Invoke-Api([string]$Method, [string]$Path, [hashtable]$Data = $null,
                    [Microsoft.PowerShell.Commands.WebRequestSession]$Session = $null,
                    [string]$Bearer = '') {
    $args = @{
        Uri = "$BaseUrl$Path"; Method = $Method; UseBasicParsing = $true; TimeoutSec = 30
    }
    if ($null -ne $Data) {
        $args.ContentType = 'application/json'
        $args.Body = ConvertTo-Json -InputObject $Data -Compress
    }
    if ($null -ne $Session) { $args.WebSession = $Session }
    if ($Bearer) { $args.Headers = @{ Authorization = "Bearer $Bearer" } }
    try {
        $r = Invoke-WebRequest @args
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
function Invoke-TestSql([string]$Sql) {
    $out = $Sql | & docker compose -f compose.yaml exec -T postgres sh -c `
        'exec psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -v ON_ERROR_STOP=1 -At'
    if ($LASTEXITCODE -ne 0) { throw 'PostgreSQL test query failed' }
    return (($out | Out-String).Trim())
}
function Get-CookieValue([Microsoft.PowerShell.Commands.WebRequestSession]$Session,
                         [string]$Name) {
    $cookie = $Session.Cookies.GetCookies([uri]$BaseUrl)[$Name]
    if ($null -eq $cookie) { return '' }
    return [string]$cookie.Value
}
function Get-VerificationMessages([string]$Address) {
    $q = [uri]::EscapeDataString("to:$Address")
    $data = Invoke-RestMethod -Uri "$MailpitUrl/api/v1/search?query=$q" -TimeoutSec 10
    return @($data.messages | Where-Object {
        $message = $_
        @($message.To | Where-Object { $_.Address -eq $Address }).Count -gt 0 -and
        ([string]$message.Subject).ToLowerInvariant().Contains('verification')
    })
}
function Wait-VerificationCode([string]$Address, [int]$PreviousCount) {
    for ($i = 0; $i -lt 40; $i++) {
        $messages = @(Get-VerificationMessages $Address)
        if ($messages.Count -gt $PreviousCount) {
            $latest = $messages | Sort-Object -Property Created -Descending | Select-Object -First 1
            $message = Invoke-RestMethod -Uri "$MailpitUrl/api/v1/message/$($latest.ID)" -TimeoutSec 10
            $match = [regex]::Match([string]$message.Text, '(?i)code is\s+(\d{6})')
            if (-not $match.Success) { throw 'Unexpected verification email format' }
            return $match.Groups[1].Value
        }
        Start-Sleep -Milliseconds 300
    }
    throw ('No email received for {0}' -f $Address)
}
function New-FullSession([string]$LoginName, [string]$EmailAddress,
                         [string]$Password, [string]$RecoveryCode) {
    $s = New-Object Microsoft.PowerShell.Commands.WebRequestSession
    $previous = @(Get-VerificationMessages $EmailAddress).Count
    $first = Invoke-Api 'Post' '/auth/login' @{ login = $LoginName; password = $Password } $s
    Assert-Status "Password accepted: $LoginName" $first 204
    Assert-True "Pending login cookie: $LoginName" ((Get-CookieValue $s 'auth_pending') -match '^[0-9a-f]{64}$')
    $code = Wait-VerificationCode $EmailAddress $previous
    $second = Invoke-Api 'Post' '/auth/login/email-code' @{ code = $code } $s
    Assert-Status "Email accepted: $LoginName" $second 204
    Assert-True "Pending recovery cookie: $LoginName" ((Get-CookieValue $s 'auth_recovery_pending') -match '^[0-9a-f]{64}$')
    $third = Invoke-Api 'Post' '/auth/login/recovery-code' @{ code = $RecoveryCode } $s
    Assert-Status "Recovery code accepted: $LoginName" $third 204
    $token = Get-CookieValue $s 'session'
    Assert-True "Full session issued: $LoginName" ($token -match '^[0-9a-f]{64}$')
    return $token
}
function Get-Principal([string]$Token) {
    $r = Invoke-Api 'Post' '/internal/introspect' @{ token = $Token }
    Assert-Status 'Introspection HTTP 200' $r 200
    return (ConvertFrom-Json -InputObject $r.Body)
}

try {
    $health = Invoke-RestMethod -Uri "$BaseUrl/health" -TimeoutSec 10
    Assert-True 'AUTH health OK' ($health.status -eq 'ok')
    $null = Invoke-RestMethod -Uri "$MailpitUrl/api/v1/info" -TimeoutSec 10
    Assert-True 'Mailpit available' $true
    Assert-True 'PostgreSQL available' ((Invoke-TestSql 'SELECT 1;') -eq '1')

    $suffix = [guid]::NewGuid().ToString('N').Substring(0, 12)
    $regularLogin = "auth29u_$suffix"
    $adminLogin = "auth29a_$suffix"
    $createdLogin = "auth29n_$suffix"
    $regularEmail = "$regularLogin@example.test"
    $adminEmail = "$adminLogin@example.test"
    $password = 'TestPassword123!'
    $newUserData = @{ login = $createdLogin; email = "$createdLogin@example.test";
                      password = $password; role = 'user' }

    $regularSignup = Invoke-Api 'Post' '/auth/signup' @{
        login = $regularLogin; email = $regularEmail; password = $password
    }
    Assert-Status 'Ordinary user registered' $regularSignup 201
    $script:createdLogins += $regularLogin
    $regularCodes = ConvertFrom-Json -InputObject $regularSignup.Body
    Assert-True 'Ordinary user has recovery codes' ($regularCodes.Count -eq 10)

    $adminSignup = Invoke-Api 'Post' '/auth/signup' @{
        login = $adminLogin; email = $adminEmail; password = $password
    }
    Assert-Status 'Admin test account registered' $adminSignup 201
    $script:createdLogins += $adminLogin
    $adminCodes = ConvertFrom-Json -InputObject $adminSignup.Body
    Assert-True 'Admin test account has recovery codes' ($adminCodes.Count -eq 10)

    $testUsers = Invoke-TestSql "SELECT count(*) FROM users WHERE login IN ('$regularLogin', '$adminLogin');"
    Assert-True 'Both HTTP users are visible in test database' ($testUsers -eq '2')

    # Test-only role escalation: only account created by this test, NEVER existing users.
    $roleChange = Invoke-TestSql @"
DELETE FROM user_roles WHERE user_id = (SELECT id FROM users WHERE login = '$adminLogin');
INSERT INTO user_roles (user_id, role_id)
SELECT u.id, r.id FROM users u CROSS JOIN roles r
WHERE u.login = '$adminLogin' AND r.code = 'admin';
"@
    Assert-True 'Test account granted admin role' ($roleChange -match 'INSERT 0 1')
    $roleCheck = Invoke-TestSql @"
SELECT r.code FROM user_roles ur JOIN users u ON u.id = ur.user_id
JOIN roles r ON r.id = ur.role_id WHERE u.login = '$adminLogin';
"@
    Assert-True 'Admin role verified in database' ($roleCheck -eq 'admin')

    # Unauthenticated calls must be rejected.
    Assert-Status 'GET /users without token' (Invoke-Api 'Get' '/users') 401
    Assert-Status 'POST /users without token' (Invoke-Api 'Post' '/users' $newUserData) 401
    Assert-Status 'GET /users with bogus token' (Invoke-Api 'Get' '/users' $null $null ('a' * 64)) 401
    Assert-Status 'POST /users with bogus token' (Invoke-Api 'Post' '/users' $newUserData $null ('a' * 64)) 401

    # Genuine 3-factor sessions for user and admin.
    $regularToken = New-FullSession $regularLogin $regularEmail $password ([string]$regularCodes[0])
    $adminToken = New-FullSession $adminLogin $adminEmail $password ([string]$adminCodes[0])
    $ordinaryPrincipal = Get-Principal $regularToken
    Assert-True 'Regular session principal has user role' ($ordinaryPrincipal.active -eq $true -and $ordinaryPrincipal.role -eq 'user')
    $adminPrincipal = Get-Principal $adminToken
    Assert-True 'Admin session principal has admin role' ($adminPrincipal.active -eq $true -and $adminPrincipal.role -eq 'admin')

    Assert-Status 'GET /users as user returns 403' (Invoke-Api 'Get' '/users' $null $null $regularToken) 403
    Assert-Status 'POST /users as user returns 403' (Invoke-Api 'Post' '/users' $newUserData $null $regularToken) 403
    $notCreated = Invoke-TestSql "SELECT count(*) FROM users WHERE login = '$createdLogin';"
    Assert-True 'Forbidden POST did not create a user' ($notCreated -eq '0')

    $adminList = Invoke-Api 'Get' '/users' $null $null $adminToken
    Assert-Status 'GET /users as admin returns 200' $adminList 200
    $entries = ConvertFrom-Json -InputObject $adminList.Body
    Assert-True 'User list is not empty' ($entries.Count -ge 2)
    $required = @('login', 'email', 'role')
    $allSafe = $true
    foreach ($entry in $entries) {
        $keys = @($entry.PSObject.Properties.Name)
        if ($keys.Count -ne 3 -or @($keys | Where-Object { $_ -notin $required }).Count -gt 0 -or
            @($required | Where-Object { $_ -notin $keys }).Count -gt 0) {
            $allSafe = $false
            break
        }
    }
    Assert-True 'Every list entry has only login, email and role' $allSafe
    $regularEntry = @($entries | Where-Object { $_.login -eq $regularLogin })
    $adminEntry = @($entries | Where-Object { $_.login -eq $adminLogin })
    Assert-True 'Ordinary user appears as user' ($regularEntry.Count -eq 1 -and $regularEntry[0].role -eq 'user')
    Assert-True 'Admin appears as admin' ($adminEntry.Count -eq 1 -and $adminEntry[0].role -eq 'admin')

    $created = Invoke-Api 'Post' '/users' $newUserData $null $adminToken
    Assert-Status 'POST /users as admin returns 201' $created 201
    # Register cleanup before any subsequent assertion.
    $script:createdLogins += $createdLogin
    $newCodes = ConvertFrom-Json -InputObject $created.Body
    Assert-True 'Admin-created user has ten recovery codes' ($newCodes.Count -eq 10)
    $newRecord = Invoke-TestSql "SELECT count(*) FROM users WHERE login = '$createdLogin';"
    Assert-True 'Admin-created user exists in database' ($newRecord -eq '1')
    $updatedList = Invoke-Api 'Get' '/users' $null $null $adminToken
    Assert-Status 'Admin can list users after creation' $updatedList 200
    $updatedRows = ConvertFrom-Json -InputObject $updatedList.Body
    Assert-True 'New account appears in admin list' (@($updatedRows | Where-Object {
        $_.login -eq $createdLogin -and $_.email -eq $newUserData.email -and $_.role -eq 'user'
    }).Count -eq 1)

    $logoutWithoutToken = Invoke-Api 'Post' '/auth/logout' @{}
    Assert-Status 'Logout without bearer token returns 401' $logoutWithoutToken 401
    $logout = Invoke-Api 'Post' '/auth/logout' @{} $null $adminToken
    Assert-Status 'Admin logout returns 204' $logout 204
    $inactive = Get-Principal $adminToken
    Assert-True 'Logged-out session is inactive' ($inactive.active -eq $false)
    Assert-Status 'Old admin token cannot GET /users' (Invoke-Api 'Get' '/users' $null $null $adminToken) 401
    Assert-Status 'Old admin token cannot POST /users' (Invoke-Api 'Post' '/users' $newUserData $null $adminToken) 401
    $stillActive = Get-Principal $regularToken
    Assert-True 'Other user session unaffected by admin logout' ($stillActive.active -eq $true)

    Write-Host ("`nAUTH-29 E2E PASSED: {0} checks" -f $script:passed)
}
catch {
    Write-Error ('AUTH-29 E2E FAILED: {0}' -f $_)
    exit 1
}
finally {
    foreach ($testLogin in $script:createdLogins) {
        try {
            $deleted = Invoke-TestSql "DELETE FROM users WHERE login = '$testLogin';"
            if ($deleted -eq 'DELETE 1') {
                Write-Host ('[CLEANUP] User {0} removed' -f $testLogin)
            } else {
                Write-Warning ('Unexpected cleanup for {0}: {1}' -f $testLogin, $deleted)
            }
        }
        catch { Write-Warning ('Failed to clean up {0}: {1}' -f $testLogin, $_) }
    }
}
