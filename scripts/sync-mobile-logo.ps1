$ErrorActionPreference = 'Stop'
$repo = Split-Path $PSScriptRoot -Parent
$source = Join-Path $repo 'app/logo.jpg'
$android = Join-Path $repo 'android/app/src/main/res/drawable-nodpi'
$catalog = Join-Path $repo 'ios/App/Assets.xcassets'
$imageSet = Join-Path $catalog 'TuiGoLogo.imageset'
$iconSet = Join-Path $catalog 'AppIcon.appiconset'
foreach ($dir in @($android,$catalog,$imageSet,$iconSet)) { New-Item -ItemType Directory -Force $dir | Out-Null }
# The exact image embedded by the Windows app; no new artwork or crop.
Copy-Item -LiteralPath $source -Destination (Join-Path $android 'tuigo_logo.jpg')
Copy-Item -LiteralPath $source -Destination (Join-Path $imageSet 'logo.jpg')
$info = @{version=1;author='xcode'}
@{info=$info} | ConvertTo-Json -Depth 4 | Set-Content -Encoding utf8 (Join-Path $catalog 'Contents.json')
@{images=@(@{idiom='universal';filename='logo.jpg';scale='1x'});info=$info} | ConvertTo-Json -Depth 5 | Set-Content -Encoding utf8 (Join-Path $imageSet 'Contents.json')
$slots = @(
    @('iphone','20x20','2x',40),@('iphone','20x20','3x',60),
    @('iphone','29x29','2x',58),@('iphone','29x29','3x',87),
    @('iphone','40x40','2x',80),@('iphone','40x40','3x',120),
    @('iphone','60x60','2x',120),@('iphone','60x60','3x',180),
    @('ipad','20x20','1x',20),@('ipad','20x20','2x',40),
    @('ipad','29x29','1x',29),@('ipad','29x29','2x',58),
    @('ipad','40x40','1x',40),@('ipad','40x40','2x',80),
    @('ipad','76x76','1x',76),@('ipad','76x76','2x',152),
    @('ipad','83.5x83.5','2x',167),@('ios-marketing','1024x1024','1x',1024)
)
Add-Type -AssemblyName System.Drawing
$original = [System.Drawing.Image]::FromFile($source)
try {
    foreach ($size in ($slots | ForEach-Object { $_[3] } | Sort-Object -Unique)) {
        $bitmap = [System.Drawing.Bitmap]::new($size,$size,[System.Drawing.Imaging.PixelFormat]::Format24bppRgb)
        $graphic = [System.Drawing.Graphics]::FromImage($bitmap)
        try {
            $graphic.InterpolationMode = [System.Drawing.Drawing2D.InterpolationMode]::HighQualityBicubic
            $graphic.DrawImage($original,0,0,$size,$size)
            $bitmap.Save((Join-Path $iconSet "icon-$size.png"),[System.Drawing.Imaging.ImageFormat]::Png)
        } finally { $graphic.Dispose(); $bitmap.Dispose() }
    }
} finally { $original.Dispose() }
$images = foreach ($slot in $slots) { @{idiom=$slot[0];size=$slot[1];scale=$slot[2];filename="icon-$($slot[3]).png"} }
@{images=@($images);info=$info} | ConvertTo-Json -Depth 5 | Set-Content -Encoding utf8 (Join-Path $iconSet 'Contents.json')
Write-Output 'Synced the Windows logo into Android and iOS assets, including opaque iOS app icons.'
