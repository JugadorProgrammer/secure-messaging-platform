
param(
    [string]$BaseUrl = "http://127.0.0.1:8080"
)

$ErrorActionPreference = "Stop"

$BaseUrl = $BaseUrl.TrimEnd("/")
$script:passed = 0

function Send-Signup {
    param([hashtable]$Data)

    $body = ConvertTo-Json -InputObject $Data -Compress

    try {
        $response = Invoke-WebRequest `
            -Uri "$BaseUrl/auth/signup" `
            -Method Post `
            -ContentType "application/json" `
            -Body $body `
            -UseBasicParsing `
            -TimeoutSec 30

        return @{
            Status = [int]$response.StatusCode
            Body = $response.Content
        }
    }
    catch [System.Net.WebException] {
        $response = $_.Exception.Response

        if ($null -eq $response) {
            throw
        }

        $status = [int]$response.StatusCode
        $reader = New-Object System.IO.StreamReader(
            $response.GetResponseStream()
        )

        try {
            $content = $reader.ReadToEnd()
        }
        finally {
            $reader.Dispose()
            $response.Dispose()
        }

        return @{
            Status = $status
            Body = $content
        }
    }
}

function Assert-Status {
    param(
        [string]$Name,
        [hashtable]$Result,
        [int]$Expected
    )

    if ($Result.Status -ne $Expected) {
        throw "$Name : expected $Expected, got $($Result.Status)"
    }

    $script:passed++
    Write-Host "[PASS] $Name"
}

try {
    $suffix = [guid]::NewGuid().ToString("N").Substring(0, 12)

    $login = "auth25_$suffix"
    $email = "$login@example.test"

    $valid = @{
        login = $login
        email = $email
        password = "TestPassword123!"
    }

    # 1. Успешная регистрация
    $result = Send-Signup $valid
    Assert-Status "Successful signup" $result 201

    $codes = ConvertFrom-Json -InputObject $result.Body

    if ($codes.Count -ne 10) {
        throw "Expected 10 recovery codes"
    }

    if (@($codes | Select-Object -Unique).Count -ne 10) {
        throw "Recovery codes are not unique"
    }

    if (@($codes | Where-Object {
        $_ -isnot [string] -or
        $_ -notmatch '^[0-9A-F]{8}(-[0-9A-F]{8}){3}$'
    }).Count -ne 0) {
        throw "Invalid recovery code format"
    }

    $script:passed++
    Write-Host "[PASS] Ten unique recovery codes"

    # 2. Повторный логин
    $duplicate = $valid.Clone()
    $duplicate.email = "other_$suffix@example.test"

    Assert-Status "Duplicate login" `
        (Send-Signup $duplicate) 409

    # 3. Логин в другом регистре
    $duplicate.login = $login.ToUpperInvariant()

    Assert-Status "Case insensitive login" `
        (Send-Signup $duplicate) 409

    # 4. Повторный email
    $duplicate = $valid.Clone()
    $duplicate.login = "other_$suffix"

    Assert-Status "Duplicate email" `
        (Send-Signup $duplicate) 409

    # 5. Email в другом регистре
    $duplicate.email = $email.ToUpperInvariant()

    Assert-Status "Case insensitive email" `
        (Send-Signup $duplicate) 409

    # 6. Некорректный логин
    $invalid = $valid.Clone()
    $invalid.login = "ab"

    Assert-Status "Invalid login" `
        (Send-Signup $invalid) 400

    # 7. Некорректный email
    $invalid = $valid.Clone()
    $invalid.email = "invalid-email"

    Assert-Status "Invalid email" `
        (Send-Signup $invalid) 400

    # 8. Некорректный пароль
    $invalid = $valid.Clone()
    $invalid.password = "short"

    Assert-Status "Invalid password" `
        (Send-Signup $invalid) 400

    # 9. Пропущен логин
    $invalid = $valid.Clone()
    $invalid.Remove("login")

    Assert-Status "Missing login" `
        (Send-Signup $invalid) 400

    # 10. Пропущен email
    $invalid = $valid.Clone()
    $invalid.Remove("email")

    Assert-Status "Missing email" `
        (Send-Signup $invalid) 400

    # 11. Пропущен пароль
    $invalid = $valid.Clone()
    $invalid.Remove("password")

    Assert-Status "Missing password" `
        (Send-Signup $invalid) 400

    Write-Host ""
    Write-Host "AUTH-25 E2E PASSED: $script:passed checks"
    exit 0
}
catch {
    Write-Error "AUTH-25 E2E FAILED: $_"
    exit 1
}
