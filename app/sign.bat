@echo off
rem sign.bat <exe> - sign with the local BoGoTiengViet Dev cert
set SIGNTOOL="C:\Program Files (x86)\Windows Kits\10\bin\10.0.26100.0\x64\signtool.exe"
set CERT_SHA1=B7577D3B0D04982B3C597885D69C292FA0801DD4
%SIGNTOOL% sign /fd SHA256 /sha1 %CERT_SHA1% /s My %1
