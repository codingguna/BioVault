$clsid = [Guid]"{7C6A8723-53BC-4545-98D1-9B5F9542D4C1}"
$type = [Type]::GetTypeFromCLSID($clsid)
$instance = [Activator]::CreateInstance($type)

$instance.SetUsageScenario(1, 0)
$cred = $null
$instance.GetCredentialAt(0, [ref]$cred)

Write-Host "Testing Mfs100Credential methods:"

# 1. GetStringValue
for ($i = 0; $i -lt 4; $i++) {
    $str = ""
    $r = $cred.GetStringValue($i, [ref]$str)
    Write-Host "GetStringValue($i): res=$r, str='$str'"
}

# 2. GetBitmapValue
$hbmp = [IntPtr]::Zero
$r = $cred.GetBitmapValue(0, [ref]$hbmp)
Write-Host "GetBitmapValue(0): res=$r, hbmp=$hbmp"

# 3. GetFieldState
for ($i = 0; $i -lt 4; $i++) {
    $state = 0
    $istate = 0
    $r = $cred.GetFieldState($i, [ref]$state, [ref]$istate)
    Write-Host "GetFieldState($i): res=$r, state=$state, istate=$istate"
}

# 4. GetSubmitButtonValue & GetComboBoxValueCount
$adj = 0
$r = $cred.GetSubmitButtonValue(0, [ref]$adj)
Write-Host "GetSubmitButtonValue(0): res=$r (expected 0x80004001 E_NOTIMPL)"

$items = 0
$sel = 0
$r = $cred.GetComboBoxValueCount(0, [ref]$items, [ref]$sel)
Write-Host "GetComboBoxValueCount(0): res=$r (expected 0x80004001 E_NOTIMPL)"
