Set-Location $PSScriptRoot/../

$packages = go list -deps ./cmd/buf | Select-String -Pattern "github.com/bufbuild/buf"

# Powershell is like ruby, the last expression in a block is yielded as a value so this
# behaves like mapping a function to the list of packages
$relativePaths = $packages | ForEach-Object {
    $package = $PSItem | Out-String
    "./" + $package.Replace("github.com/bufbuild/buf/", "").Trim()
}

go test $relativePaths