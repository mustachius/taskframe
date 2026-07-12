# install.ps1 — compila o taskframe e instala como comando global do usuário.
# Depois disso, `taskframe` funciona de qualquer pasta no PowerShell, como o
# `claude`. Não precisa de admin (instala só para o seu usuário).
#
#   uso:  .\install.ps1

$ErrorActionPreference = 'Stop'
$root = $PSScriptRoot

# 1. Go precisa estar disponível
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Error "Go não encontrado no PATH. Instale com: winget install GoLang.Go"
    exit 1
}

# 2. pasta de destino, por-usuário (sem admin)
$target = Join-Path $env:LOCALAPPDATA 'Programs\taskframe'
New-Item -ItemType Directory -Force -Path $target | Out-Null
$exe = Join-Path $target 'taskframe.exe'

# 3. compilar para um nome temporário (sobrevive a uma instância em execução)
Write-Host "compilando..." -ForegroundColor Cyan
$env:CGO_ENABLED = '0'
$new = "$exe.new"
& go build -o $new (Join-Path $root 'cmd\taskframe')
if ($LASTEXITCODE -ne 0) { Write-Error "falha na compilação"; exit 1 }

# 4. trocar o exe (rename de um exe em uso é permitido no Windows)
if (Test-Path $exe) {
    $stamp = Get-Date -Format 'yyyyMMddHHmmss'
    Rename-Item -Path $exe -NewName "taskframe.exe.old-$stamp" -Force
}
Move-Item -Path $new -Destination $exe -Force
# limpar versões antigas (best-effort: uma ainda em uso é ignorada)
Get-ChildItem -Path $target -Filter 'taskframe.exe.old-*' -ErrorAction SilentlyContinue |
    ForEach-Object { try { Remove-Item $_.FullName -Force -ErrorAction Stop } catch {} }

# 5. adicionar ao PATH do usuário, de forma idempotente
$added = $false
try {
    $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    $parts = @($userPath -split ';' | Where-Object { $_ -ne '' })
    $norm = { param($p) $p.TrimEnd('\').ToLowerInvariant() }
    $has = $parts | Where-Object { (& $norm $_) -eq (& $norm $target) }
    if (-not $has) {
        [Environment]::SetEnvironmentVariable('Path', (($parts + $target) -join ';'), 'User')
        $env:Path += ';' + $target   # vale já nesta sessão
        $added = $true
    }
} catch {
    Write-Warning "não consegui editar o PATH do usuário automaticamente."
    Write-Warning "adicione manualmente esta pasta ao PATH: $target"
}

Write-Host ""
Write-Host "instalado em: $exe" -ForegroundColor Green
if ($added) {
    Write-Host "PATH atualizado. Abra um NOVO terminal e rode: " -NoNewline
    Write-Host "taskframe" -ForegroundColor Yellow
} else {
    Write-Host "já estava no PATH. Rode: " -NoNewline
    Write-Host "taskframe" -ForegroundColor Yellow
}
