#!/bin/bash

# Exit immediately if a command exits with a non-zero status
set -e

# Application details
APP_NAME="Crop-Disease-Predictor"
BUILD_DIR="build"
INSTALL_DIR="/usr/local/bin"
ICON_PATH="$BUILD_DIR/assets/icon.png"
DESKTOP_FILE="/usr/share/applications/$APP_NAME.desktop"

# Check for root privileges
if [ "$EUID" -ne 0 ]; then
    echo "Please run as root to install the application."
    exit 1
fi

# Install the executable
if [ -f "$BUILD_DIR/$APP_NAME" ]; then
    echo "Installing the application..."
    cp "$BUILD_DIR/$APP_NAME" "$INSTALL_DIR"
    chmod +x "$INSTALL_DIR/$APP_NAME"
    echo "Executable installed to $INSTALL_DIR/$APP_NAME"
else
    echo "Error: Executable not found in $BUILD_DIR. Run the build script first."
    exit 1
fi

# Install the icon
if [ -f "$ICON_PATH" ]; then
    echo "Installing application icon..."
    cp "$ICON_PATH" /usr/share/icons/hicolor/256x256/apps/$APP_NAME.png
    echo "Icon installed to /usr/share/icons/hicolor/256x256/apps/$APP_NAME.png"
else
    echo "Warning: Icon not found at $ICON_PATH. Skipping icon installation."
fi

# Create a .desktop file
echo "Creating desktop entry..."
cat <<EOF > "$DESKTOP_FILE"
[Desktop Entry]
Version=1.0
Type=Application
Name=$APP_NAME
Exec=$INSTALL_DIR/$APP_NAME
Icon=$APP_NAME
Terminal=false
Categories=Utility;
EOF
chmod 644 "$DESKTOP_FILE"
echo "Desktop entry created at $DESKTOP_FILE"

# Update desktop database
echo "Updating desktop database..."
update-desktop-database /usr/share/applications

echo "Installation completed successfully."
