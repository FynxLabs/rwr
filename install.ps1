# Define default paths
$BINARY_PATH = "$env:ProgramFiles\rwr"
$LICENSE_PATH = "$env:ProgramFiles\rwr\doc"
$README_PATH = "$env:ProgramFiles\rwr\doc"

# GitHub repository owner and name
$REPO = "FynxLabs/rwr"

# Detect operating system
$OS = "Windows"

# Detect architecture
$ARCH = if ([Environment]::Is64BitOperatingSystem) { "x86_64" } else { "i386" }

# Get the latest release data from the GitHub API
$latest_release = Invoke-RestMethod -Uri "https://api.github.com/repos/$REPO/releases/latest"

# Extract the download URL for the desired asset
$download_url = $latest_release.assets |
    Where-Object { $_.name -like "rwr_${OS}_${ARCH}.zip" } |
    Select-Object -ExpandProperty browser_download_url

if (-not $download_url) {
    Write-Host "Could not find a download URL for $OS $ARCH. Exiting."
    exit 1
}

# Download the file
$tmp_file = "$env:TEMP\rwr.zip"
Invoke-WebRequest -Uri $download_url -OutFile $tmp_file

# Extract the zip file to a temporary directory
$tmp_extract = "$env:TEMP\rwr_extracted"
Expand-Archive -Path $tmp_file -DestinationPath $tmp_extract -Force

# Create the installation directory if it doesn't exist
New-Item -ItemType Directory -Force -Path $BINARY_PATH | Out-Null
New-Item -ItemType Directory -Force -Path $LICENSE_PATH | Out-Null
New-Item -ItemType Directory -Force -Path $README_PATH | Out-Null

# Move the binary to the default binary path
Move-Item -Path "$tmp_extract\rwr.exe" -Destination $BINARY_PATH -Force

# Move the LICENSE and README to the default documentation path
Move-Item -Path "$tmp_extract\LICENSE" -Destination $LICENSE_PATH -Force
Move-Item -Path "$tmp_extract\README" -Destination $README_PATH -Force

# Add the binary path to system PATH if it's not already there
$current_path = [Environment]::GetEnvironmentVariable("Path", "Machine")
if (-not $current_path.Split(';').Contains($BINARY_PATH)) {
    $new_path = $current_path + ";" + $BINARY_PATH
    [Environment]::SetEnvironmentVariable("Path", $new_path, "Machine")
}

# Clean up temporary files
Remove-Item -Path $tmp_file -Force
Remove-Item -Path $tmp_extract -Recurse -Force

Write-Host "rwr has been installed successfully for $OS $ARCH."