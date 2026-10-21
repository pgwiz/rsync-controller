@echo off
echo ========================================================
echo  Rsync Controller - Windows Service One-Click Uninstaller
echo ========================================================
echo Requesting Administrator elevation to remove service...
powershell -NoProfile -Command "Start-Process -FilePath '%~dp0rsync-service.exe' -ArgumentList '--uninstall' -Verb RunAs"
echo.
echo Process complete.
pause
