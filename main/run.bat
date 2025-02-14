@echo off
REM -----------------------------------------------------------------------------
REM run.bat
REM This batch file uses template placeholders that should be replaced by the
REM Go program before writing it to disk.
REM Replace the following placeholders:
REM   {{PYTHON_EXE}}   - Absolute path to the Python executable.
REM   {{MAIN_SCRIPT}}  - Absolute path to the main Python script.
REM   {{SCRIPTS_DIR}}  - Absolute path to the scripts directory.
REM -----------------------------------------------------------------------------

REM Convert the placeholders to absolute paths (if they are relative).
FOR %%I in ("{{PYTHON_EXE}}") DO set "PYTHON_EXE=%%~fI"
FOR %%I in ("{{MAIN_SCRIPT}}") DO set "MAIN_SCRIPT=%%~fI"
FOR %%I in ("{{SCRIPTS_DIR}}") DO set "SCRIPTS_DIR=%%~fI"

REM Convert backslashes to forward slashes for Python compatibility.
set "SCRIPTS_DIR_MOD=%SCRIPTS_DIR:\=/%"
set "MAIN_SCRIPT_MOD=%MAIN_SCRIPT:\=/%"

REM Set the PYTHONPATH environment variable.
set "PYTHONPATH=%SCRIPTS_DIR%"

REM Construct the inline Python code.
set "PY_CODE=import sys; sys.path.insert(0, '%SCRIPTS_DIR_MOD%'); exec(open('%MAIN_SCRIPT_MOD%').read())"

REM Execute the Python command.
"%PYTHON_EXE%" -c "%PY_CODE%"

if errorlevel 1 (
    echo Error: Python script execution failed.
    exit /b 1
)

pause
