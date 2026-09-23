$cert = New-SelfSignedCertificate -Subject 'CN=BoGoTiengViet Dev' -Type CodeSigningCert -CertStoreLocation 'Cert:\CurrentUser\My' -NotAfter (Get-Date).AddYears(5) -KeyExportPolicy Exportable
foreach ($store in 'Root','TrustedPublisher') {
    $s = New-Object System.Security.Cryptography.X509Certificates.X509Store($store,'CurrentUser')
    $s.Open('ReadWrite')
    $s.Add($cert)
    $s.Close()
}
Write-Output $cert.Thumbprint
