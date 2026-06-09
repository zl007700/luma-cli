@echo off
setlocal

cd /d "%~dp0\.."

set "VERSION=local"

REM Auto-derive version from the most recent git tag (e.g. "0.0.19-11-g654a5b7")
REM so the output reflects how far this build is from the last release. The
REM leading "v" is stripped to match the package.json convention. Falls back
REM silently to "local" when not in a git checkout or no annotated tags exist;
REM any positional argument below overrides this.
for /f "usebackq delims=" %%i in (`git describe --tags --long 2^>nul`) do set "VERSION=%%i"
if "%VERSION%"=="" set "VERSION=local"
if "%VERSION:~0,1%"=="v" set "VERSION=%VERSION:~1%"

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

REM Capture build metadata. Fall back to "unknown" when git is unavailable
REM or the working tree is not a checkout.
for /f "usebackq delims=" %%i in (`git rev-parse --short HEAD 2^>nul`) do set "COMMIT=%%i"
if "%COMMIT%"=="" set "COMMIT=unknown"
for /f "usebackq tokens=*" %%i in (`powershell -NoProfile -Command "(Get-Date).ToUniversalTime().ToString('yyyy-MM-dd')"`) do set "BUILD_DATE=%%i"
if "%BUILD_DATE%"=="" set "BUILD_DATE=unknown"

set "OUT_DIR=build\local"
set "OUT=%OUT_DIR%\luma-cli.exe"
set "LDFLAGS=-X github.com/luma-cli/lumer-cli/internal/commands.version=%VERSION% -X github.com/luma-cli/lumer-cli/internal/commands.commit=%COMMIT% -X github.com/luma-cli/lumer-cli/internal/commands.buildDate=%BUILD_DATE%"

if not exist "%OUT_DIR%" mkdir "%OUT_DIR%"

echo Building luma-cli %VERSION% (commit %COMMIT%, built %BUILD_DATE%)...
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
