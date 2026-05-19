@echo off
setlocal

cd /d "%~dp0\.."

set "VERSION=local"
set "INSTALL=0"

:parse
if "%~1"=="" goto parsed
if /I "%~1"=="--install" (
  set "INSTALL=1"
  shift
  goto parse
)
if /I "%~1"=="install" (
  set "INSTALL=1"
  shift
  goto parse
)
set "VERSION=%~1"
shift
goto parse

:parsed
set "OUT_DIR=build\local"
set "OUT=%OUT_DIR%\luma-cli.exe"
set "LDFLAGS=-X github.com/luma-cli/lumer-cli/internal/commands.version=%VERSION%"

if not exist "%OUT_DIR%" mkdir "%OUT_DIR%"

echo Building luma-cli %VERSION%...
go build -ldflags "%LDFLAGS%" -o "%OUT%" .
if errorlevel 1 exit /b %errorlevel%

echo Built: %CD%\%OUT%

if "%INSTALL%"=="0" goto done

for /f "usebackq delims=" %%i in (`npm root -g`) do set "NPM_ROOT=%%i"
if "%NPM_ROOT%"=="" (
  echo Error: unable to resolve npm global root. Is npm installed?
  exit /b 1
)

set "TARGET_DIR=%NPM_ROOT%\@lumageo\luma-cli\bin"
set "TARGET=%TARGET_DIR%\luma-cli.exe"

if not exist "%TARGET_DIR%" mkdir "%TARGET_DIR%"
copy /Y "%OUT%" "%TARGET%" >nul
if errorlevel 1 exit /b %errorlevel%

echo Installed local binary: %TARGET%
echo Try: luma-cli version

:done
endlocal
