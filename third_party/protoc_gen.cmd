@echo off
setlocal

:: Переходим в корень проекта
cd /D %~dp0..

:: Проверяем существование proto-файла
if not exist "api\supervisor.proto" (
    echo Error: api\supervisor.proto not found!
    exit /b 1
)

:: Запускаем генерацию
protoc --go_out=api/gosv --go_opt=paths=source_relative --go-grpc_out=api/gosv --go-grpc_opt=paths=source_relative api\supervisor.proto

if %errorlevel% equ 0 (
    echo Code generated successfully!
) else (
    echo Code generation failed!
)

endlocal