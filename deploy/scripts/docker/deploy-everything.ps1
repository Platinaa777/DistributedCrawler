param(
    [string]$Registry = "distributed-crawler",
    [string]$Tag = "latest",
    [string]$PgUser = "crawler",
    [string]$PgPassword = "some-pwd-123",
    [string]$PgDatabase = "crawler",
    [string]$PgPort = "54322",
    [string]$RabbitMqUser = "guest",
    [string]$RabbitMqPassword = "guest",
    [string]$MinioUser = "minioadmin",
    [string]$MinioPassword = "minioadmin",
    [string]$MinioBucket = "pages",
    [string]$RedisPassword = "some_redis_pwd_123",
    [string]$GrafanaUser = "admin",
    [string]$GrafanaPassword = "changeme-grafana-password",
    [string]$JwtSecret = "your-secret-key-change-this-in-production-make-it-long-and-random",
    [string]$DefaultUserEmail = "admin@example.com",
    [string]$DefaultUserPassword = "12345678",
    [string]$MessagingBroker = "rabbitmq",
    [string]$CorsOrigin = "http://localhost:4200",
    [string]$QueueSecretsFile = "",
    [switch]$AppOnly,
    [switch]$NoBuild,
    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$ComposeArgs
)

$ErrorActionPreference = "Stop"

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$launchScript = Join-Path $scriptDir "launch.sh"
$projectRoot = Split-Path -Parent (Split-Path -Parent (Split-Path -Parent $scriptDir))

if ([string]::IsNullOrWhiteSpace($QueueSecretsFile)) {
    $QueueSecretsFile = Join-Path $projectRoot "queue-secrets.json.example"
}

$bashCommand = Get-Command bash -ErrorAction SilentlyContinue
if (-not $bashCommand) {
    throw "bash was not found in PATH. Install Git Bash or WSL, or run deploy/scripts/docker/deploy-everything.sh from bash."
}

$argsList = @(
    $launchScript,
    "--registry", $Registry,
    "--tag", $Tag,
    "--pg-user", $PgUser,
    "--pg-password", $PgPassword,
    "--pg-database", $PgDatabase,
    "--pg-port", $PgPort,
    "--rabbitmq-user", $RabbitMqUser,
    "--rabbitmq-password", $RabbitMqPassword,
    "--minio-user", $MinioUser,
    "--minio-password", $MinioPassword,
    "--minio-bucket", $MinioBucket,
    "--redis-password", $RedisPassword,
    "--grafana-user", $GrafanaUser,
    "--grafana-password", $GrafanaPassword,
    "--jwt-secret", $JwtSecret,
    "--default-user-email", $DefaultUserEmail,
    "--default-user-password", $DefaultUserPassword,
    "--messaging-broker", $MessagingBroker,
    "--cors-origin", $CorsOrigin,
    "--queue-secrets-file", $QueueSecretsFile
)

if ($AppOnly) {
    $argsList += "--app-only"
}

if ($NoBuild) {
    $argsList += "--no-build"
}

foreach ($composeArg in $ComposeArgs) {
    $argsList += "--compose-arg"
    $argsList += $composeArg
}

& $bashCommand.Source @argsList
