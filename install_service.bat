@echo off
echo ========================================================
echo  Rsync Controller - Windows Service One-Click Installer
echo ========================================================
echo Requesting Administrator elevation...
powershell -NoProfile -Command "Start-Process -FilePath '%~dp0rsync-service.exe' -ArgumentList '--install' -Verb RunAs -Wait; Start-Service RsyncControllerDaemon -ErrorAction SilentlyContinue"
echo.
echo Process complete. Checking service status:
sc.exe query RsyncControllerDaemon
echo.
pause
