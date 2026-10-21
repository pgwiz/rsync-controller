@echo off
title Rsync Context Menu Fix
echo ========================================================
echo  Rsync Controller - Administrator Shell Menu Fix
echo ========================================================
echo Requesting Administrator elevation...
powershell -NoProfile -ExecutionPolicy Bypass -Command "Start-Process powershell -ArgumentList '-NoProfile -ExecutionPolicy Bypass -File \"\"%~dp0scripts\fix_shell_menus.ps1\"\"' -Verb RunAs"
echo.
