# Сохраните как third_party/protoc_gen.ps1
$projectRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $projectRoot

if (-not (Test-Path "api\supervisor.proto")) {
    Write-Error "api\supervisor.proto not found!"
    exit 1
}

protoc --go_out=. --go_opt=paths=source_relative `
       --go-grpc_out=. --go-grpc_opt=paths=source_relative `
       api\supervisor.proto

if ($LASTEXITCODE -eq 0) {
    Write-Host "Code generated successfully!" -ForegroundColor Green
} else {
    Write-Host "Code generation failed!" -ForegroundColor Red
}