#!/bin/bash

# ShipIt Client Release Script
# This script builds and packages the ShipIt client for distribution

set -e

# Configuration
VERSION=${VERSION:-$(git describe --tags --always --dirty)}
BINARY_NAME="shipitd"
DIST_DIR="dist"
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Clean previous builds
clean() {
    log_info "Cleaning previous builds..."
    rm -rf ${DIST_DIR}
    rm -f ${BINARY_NAME}
    rm -f ${BINARY_NAME}.exe
    rm -f ${BINARY_NAME}.linux
    rm -f ${BINARY_NAME}.darwin
    rm -f ${BINARY_NAME}.darwin-arm64
    go clean
}

# Build for all platforms
build_all() {
    log_info "Building for all platforms..."
    
    # Build flags
    LDFLAGS="-ldflags \"-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}\""
    
    # Linux
    log_info "Building for Linux..."
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o ${BINARY_NAME}.linux ./cmd/client
    
    # macOS Intel
    log_info "Building for macOS (Intel)..."
    CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build ${LDFLAGS} -o ${BINARY_NAME}.darwin ./cmd/client
    
    # macOS Apple Silicon
    log_info "Building for macOS (Apple Silicon)..."
    CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build ${LDFLAGS} -o ${BINARY_NAME}.darwin-arm64 ./cmd/client
    
    # Windows
    log_info "Building for Windows..."
    CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build ${LDFLAGS} -o ${BINARY_NAME}.exe ./cmd/client
    
    # Local build
    log_info "Building for local platform..."
    go build ${LDFLAGS} -o ${BINARY_NAME} ./cmd/client
}

# Create distribution directory
create_dist() {
    log_info "Creating distribution directory..."
    mkdir -p ${DIST_DIR}
}

# Package for macOS
package_macos() {
    log_info "Packaging for macOS..."
    
    # Create macOS package structure
    mkdir -p ${DIST_DIR}/macos
    cp ${BINARY_NAME}.darwin ${DIST_DIR}/macos/
    cp ${BINARY_NAME}.darwin-arm64 ${DIST_DIR}/macos/
    
    # Create universal binary
    log_info "Creating universal binary..."
    lipo -create -output ${DIST_DIR}/macos/${BINARY_NAME} \
        ${BINARY_NAME}.darwin \
        ${BINARY_NAME}.darwin-arm64
    
    # Create installer package
    log_info "Creating macOS installer package..."
    pkgbuild --root ${DIST_DIR}/macos \
        --identifier com.unownone.shipitd \
        --version ${VERSION} \
        --install-location /usr/local/bin \
        ${DIST_DIR}/shipitd-${VERSION}-macos.pkg
    
    # Create Homebrew formula
    create_homebrew_formula
}

# Create Homebrew formula
create_homebrew_formula() {
    log_info "Creating Homebrew formula..."
    
    cat > ${DIST_DIR}/shipitd.rb << EOF
class Shipitd < Formula
  desc "ShipIt Client Daemon - Expose local services to the internet"
  homepage "https://github.com/unownone/shipitd"
  version "${VERSION}"
  
  if OS.mac?
    if Hardware::CPU.arm?
      url "https://github.com/unownone/shipitd/releases/download/v${VERSION}/shipitd-${VERSION}-darwin-arm64.tar.gz"
      sha256 "$(shasum -a 256 ${DIST_DIR}/macos/${BINARY_NAME} | cut -d' ' -f1)"
    else
      url "https://github.com/unownone/shipitd/releases/download/v${VERSION}/shipitd-${VERSION}-darwin-amd64.tar.gz"
      sha256 "$(shasum -a 256 ${DIST_DIR}/macos/${BINARY_NAME} | cut -d' ' -f1)"
    end
  elsif OS.linux?
    url "https://github.com/unownone/shipitd/releases/download/v${VERSION}/shipitd-${VERSION}-linux-amd64.tar.gz"
    sha256 "$(shasum -a 256 ${BINARY_NAME}.linux | cut -d' ' -f1)"
  end
  
  def install
    bin.install "${BINARY_NAME}"
  end
  
  test do
    system "#{bin}/${BINARY_NAME}", "--version"
  end
end
EOF
}

# Package for Linux
package_linux() {
    log_info "Packaging for Linux..."
    
    # Create Linux package structure
    mkdir -p ${DIST_DIR}/linux
    cp ${BINARY_NAME}.linux ${DIST_DIR}/linux/
    
    # Create .deb package
    create_deb_package
    
    # Create .rpm package
    create_rpm_package
}

# Create .deb package
create_deb_package() {
    log_info "Creating .deb package..."
    
    # Create package structure
    PACKAGE_DIR=${DIST_DIR}/deb/shipitd-${VERSION}
    mkdir -p ${PACKAGE_DIR}/DEBIAN
    mkdir -p ${PACKAGE_DIR}/usr/local/bin
    mkdir -p ${PACKAGE_DIR}/etc/systemd/system
    
    # Copy binary
    cp ${BINARY_NAME}.linux ${PACKAGE_DIR}/usr/local/bin/${BINARY_NAME}
    chmod +x ${PACKAGE_DIR}/usr/local/bin/${BINARY_NAME}
    
    # Create control file
    cat > ${PACKAGE_DIR}/DEBIAN/control << EOF
Package: shipitd
Version: ${VERSION}
Section: net
Priority: optional
Architecture: amd64
Depends: systemd
Maintainer: ShipIt Team <team@shipit.dev>
Description: ShipIt Client Daemon
 Expose local services to the internet through secure tunnels.
EOF
    
    # Create systemd service file
    cat > ${PACKAGE_DIR}/etc/systemd/system/shipitd.service << EOF
[Unit]
Description=ShipIt Client Daemon
After=network.target

[Service]
Type=simple
User=shipit
ExecStart=/usr/local/bin/${BINARY_NAME} start
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF
    
    # Build .deb package
    dpkg-deb --build ${PACKAGE_DIR} ${DIST_DIR}/shipitd-${VERSION}-linux-amd64.deb
}

# Create .rpm package
create_rpm_package() {
    log_info "Creating .rpm package..."
    
    # Create package structure
    PACKAGE_DIR=${DIST_DIR}/rpm/shipitd-${VERSION}
    mkdir -p ${PACKAGE_DIR}/usr/local/bin
    mkdir -p ${PACKAGE_DIR}/etc/systemd/system
    
    # Copy binary
    cp ${BINARY_NAME}.linux ${PACKAGE_DIR}/usr/local/bin/${BINARY_NAME}
    chmod +x ${PACKAGE_DIR}/usr/local/bin/${BINARY_NAME}
    
    # Create spec file
    cat > ${DIST_DIR}/shipitd.spec << EOF
Name: shipitd
Version: ${VERSION}
Release: 1
Summary: ShipIt Client Daemon
License: MIT
URL: https://github.com/unownone/shipitd
Source0: shipitd-${VERSION}-linux-amd64.tar.gz

%description
ShipIt Client Daemon - Expose local services to the internet through secure tunnels.

%files
/usr/local/bin/${BINARY_NAME}
/etc/systemd/system/shipitd.service

%post
systemctl daemon-reload

%preun
systemctl stop shipitd || true

%postun
systemctl daemon-reload
EOF
    
    # Build .rpm package (requires rpmbuild)
    if command -v rpmbuild &> /dev/null; then
        rpmbuild -bb ${DIST_DIR}/shipitd.spec
        cp ~/rpmbuild/RPMS/x86_64/shipitd-${VERSION}-1.x86_64.rpm ${DIST_DIR}/
    else
        log_warn "rpmbuild not found, skipping .rpm package"
    fi
}

# Package for Windows
package_windows() {
    log_info "Packaging for Windows..."
    
    # Create Windows package structure
    mkdir -p ${DIST_DIR}/windows
    cp ${BINARY_NAME}.exe ${DIST_DIR}/windows/
    
    # Create installer script
    cat > ${DIST_DIR}/windows/install.bat << EOF
@echo off
echo Installing ShipIt Client...
copy ${BINARY_NAME}.exe C:\\Windows\\System32\\
echo Installation complete!
EOF
}

# Create release archive
create_archive() {
    log_info "Creating release archives..."
    
    # Create tar.gz for each platform
    cd ${DIST_DIR}
    
    # macOS
    tar -czf shipitd-${VERSION}-macos.tar.gz macos/
    
    # Linux
    tar -czf shipitd-${VERSION}-linux-amd64.tar.gz linux/
    
    # Windows
    tar -czf shipitd-${VERSION}-windows-amd64.tar.gz windows/
    
    cd ..
}

# Run tests
run_tests() {
    log_info "Running tests..."
    go test -v ./...
}

# Main function
main() {
    log_info "Starting release process for version ${VERSION}"
    
    # Run tests first
    run_tests
    
    # Clean previous builds
    clean
    
    # Build for all platforms
    build_all
    
    # Create distribution directory
    create_dist
    
    # Package for each platform
    package_macos
    package_linux
    package_windows
    
    # Create release archives
    create_archive
    
    log_info "Release complete! Distribution files are in ${DIST_DIR}/"
    log_info "Version: ${VERSION}"
    log_info "Build time: ${BUILD_TIME}"
}

# Run main function
main "$@" 