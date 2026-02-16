{ pkgs ? import <nixpkgs> {} }:

pkgs.mkShell {
  buildInputs = with pkgs; [
    pkg-config
    glib
    gtk3
    webkitgtk_4_1
    libsoup_3
    openssl
    cairo
    pango
    gdk-pixbuf
    atk
    librsvg
  ];

  shellHook = ''
    export GIO_MODULE_DIR="${pkgs.glib-networking}/lib/gio/modules"
    export XDG_DATA_DIRS="$GSETTINGS_SCHEMAS_PATH"
  '';
}
