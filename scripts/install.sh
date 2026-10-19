#!/bin/sh
# Per-user Linux/macOS installer. No administrator privileges or application runtime needed.
set -eu

say() { printf '%s\n' "$*"; }
die() { say "Error: $*" >&2; exit 1; }
usage() {
  say 'Usage: sh install.sh [--binary PATH] [--version vX.Y.Z] [--data-dir PATH] [--no-start] [--uninstall [--purge]]'
  say 'Installs Localert for the current user and enables desktop-login autostart.'
  say 'Uninstall keeps the database, icons, and logs. LOCALERT_INSTALL_DIR overrides ~/.local/bin.'
}

binary=''
data_dir=''
start=1
uninstall=0
purge=0
release_version=''
download_dir=''
temporary=''
trap 'if [ -n "$temporary" ]; then rm -f "$temporary"; fi; if [ -n "$download_dir" ]; then rm -rf "$download_dir"; fi' 0
while [ "$#" -gt 0 ]; do
  case "$1" in
    --binary) [ "$#" -ge 2 ] || die '--binary requires a path'; binary=$2; shift 2 ;;
    --version) [ "$#" -ge 2 ] || die '--version requires a tag'; release_version=$2; shift 2 ;;
    --data-dir) [ "$#" -ge 2 ] || die '--data-dir requires a path'; data_dir=$2; shift 2 ;;
    --no-start) start=0; shift ;;
    --uninstall) uninstall=1; shift ;;
    --purge) purge=1; shift ;;
    --help|-h) usage; exit 0 ;;
    *) die "Unknown argument: $1" ;;
  esac
done

[ -n "${HOME:-}" ] || die 'HOME must be set.'
uid=$(id -u)
[ "$uid" -ne 0 ] || die 'Run this installer as the desktop user, without sudo.'
case "$(uname -s)" in
  Linux) os=linux ;;
  Darwin) os=macos ;;
  *) die 'This script supports Linux and macOS. Use install.ps1 on Windows.' ;;
esac
case "$(uname -m)" in
  x86_64|amd64) arch=amd64 ;;
  aarch64|arm64) arch=arm64 ;;
  *) die 'Supported architectures are amd64 and arm64.' ;;
esac

script_dir=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd -P)
install_dir=${LOCALERT_INSTALL_DIR:-"$HOME/.local/bin"}
case "$install_dir" in /*) ;; *) install_dir="$PWD/$install_dir" ;; esac
target="$install_dir/localert"
if [ "$os" = linux ]; then
  config_home=${XDG_CONFIG_HOME:-"$HOME/.config"}
  support_dir="${XDG_DATA_HOME:-"$HOME/.local/share"}/localert"
  log_dir="${XDG_STATE_HOME:-"$HOME/.local/state"}/localert"
  autostart="$config_home/autostart/localert.desktop"
  [ -n "$data_dir" ] || data_dir="$config_home/localert"
else
  support_dir="$HOME/Library/Application Support/localert/install"
  log_dir="$HOME/Library/Logs/Localert"
  autostart="$HOME/Library/LaunchAgents/dev.localert.agent.plist"
  [ -n "$data_dir" ] || data_dir="$HOME/Library/Application Support/localert"
fi
case "$data_dir" in /*) ;; *) data_dir="$PWD/$data_dir" ;; esac
launcher="$support_dir/launch.sh"
pid_file="$log_dir/localert.pid"
log_file="$log_dir/localert.log"
case "$target$data_dir$launcher$autostart$log_file" in *'
'*) die 'Installation paths must not contain line breaks.' ;; esac

quote_sh() { printf "'%s'" "$(printf '%s' "$1" | sed "s/'/'\\\\''/g")"; }
xml_escape() { printf '%s' "$1" | sed 's/\&/\&amp;/g; s/</\&lt;/g; s/>/\&gt;/g; s/"/\&quot;/g; s/'"'"'/\&apos;/g'; }
desktop_escape() { printf '%s' "$1" | sed 's/\\/\\\\\\\\/g; s/"/\\\\"/g; s/`/\\\\`/g; s/\$/\\\\$/g; s/%/%%/g'; }

stop_existing() {
  if [ "$os" = macos ]; then
    if command -v launchctl >/dev/null 2>&1; then launchctl bootout "gui/$uid/dev.localert.agent" >/dev/null 2>&1 || true; fi
  fi
  if [ -f "$pid_file" ]; then
    pid=$(tr -d '\r\n' < "$pid_file")
    case "$pid" in ''|*[!0-9]*) return ;; esac
    command_line=$(ps -p "$pid" -o command= 2>/dev/null || true)
    case "$command_line" in
      "$target"|"$target "*)
        kill "$pid" 2>/dev/null || true
        attempts=0
        while kill -0 "$pid" 2>/dev/null; do
          case "$(ps -p "$pid" -o stat= 2>/dev/null || true)" in *Z*) break ;; esac
          [ "$attempts" -lt 200 ] || die 'Localert is still stopping. Try again shortly.'
          sleep 0.1
          attempts=$((attempts + 1))
        done
        ;;
    esac
  fi
}

if [ "$uninstall" -eq 1 ]; then
  stop_existing
  rm -f "$autostart" "$launcher" "$target" "$pid_file"
  say 'Localert and its login autostart entry were removed.'
  if [ "$purge" -eq 1 ]; then
    # Remove only files Localert creates, never a whole user-chosen folder.
    rm -f "$data_dir/localert.db" "$data_dir/localert.db-wal" "$data_dir/localert.db-shm" "$data_dir/state.json"
    rm -rf "$data_dir/icons"
    rm -f "$log_dir/localert.log" "$log_dir/localert.log.1" "$log_dir/crash.log" "$log_dir/crash.log.1"
    rmdir "$data_dir" "$log_dir" "$support_dir" 2>/dev/null || true
    say 'Your database, app keys, history, icons, and logs were deleted too.'
  else
    say "Your database and logs were kept in $data_dir. Run again with --uninstall --purge to delete them."
  fi
  exit 0
fi

if [ -z "$binary" ] && [ -z "$release_version" ] && [ -f "$0" ]; then
  for candidate in "$script_dir/localert-$os-$arch" "$script_dir/../bin/localert-$os-$arch" "$script_dir/localert" "$script_dir/../bin/localert"; do
    if [ -f "$candidate" ]; then binary=$candidate; break; fi
  done
fi
if [ -z "$binary" ]; then
  command -v curl >/dev/null 2>&1 || die 'curl is required to download Localert.'
  command -v sha256sum >/dev/null 2>&1 || command -v shasum >/dev/null 2>&1 || die 'sha256sum or shasum is required.'
  umask 077
  download_dir=$(mktemp -d "${TMPDIR:-/tmp}/localert-install.XXXXXXXX")
  api='https://api.github.com/repos/TrueFix/localert/releases/latest'
  if [ -z "$release_version" ]; then
    release_version=$(curl --fail --silent --show-error --location --proto '=https' --proto-redir '=https' --connect-timeout 15 --max-time 60 "$api" | sed -n 's/^[[:space:]]*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p')
  fi
  case "$release_version" in v[0-9]*) ;; *) die 'Cannot determine a release version; publish a GitHub release first.' ;; esac
  case "$release_version" in *[!a-zA-Z0-9.+-]*) die 'Invalid version tag.' ;; esac
  asset="localert-$os-$arch"
  base="https://github.com/TrueFix/localert/releases/download/$release_version"
  say "Downloading $release_version for $os/$arch…"
  curl --fail --silent --show-error --location --proto '=https' --proto-redir '=https' --connect-timeout 15 --max-time 180 "$base/$asset" -o "$download_dir/$asset"
  curl --fail --silent --show-error --location --proto '=https' --proto-redir '=https' --connect-timeout 15 --max-time 60 "$base/SHA256SUMS" -o "$download_dir/SHA256SUMS"
  expected=$(awk -v name="$asset" '$2 == name {print $1}' "$download_dir/SHA256SUMS")
  [ "${#expected}" -eq 64 ] || die 'Release checksum is missing or invalid.'
  if command -v sha256sum >/dev/null 2>&1; then actual=$(sha256sum "$download_dir/$asset" | awk '{print $1}'); else actual=$(shasum -a 256 "$download_dir/$asset" | awk '{print $1}'); fi
  [ "$actual" = "$expected" ] || die 'Checksum mismatch. The existing installation was not changed.'
  binary="$download_dir/$asset"
fi
[ -n "$binary" ] && [ -f "$binary" ] || die "Binary not found. Supply --binary /path/to/localert-$os-$arch."
umask 077
mkdir -p "$install_dir" "$support_dir" "$log_dir" "$data_dir" "$(dirname -- "$autostart")"
temporary="$install_dir/.localert-install-$$"
cp "$binary" "$temporary"
chmod 755 "$temporary"
if ! binary_version=$("$temporary" --version); then die 'The binary could not run on this OS/architecture.'; fi
case "$binary_version" in 'Localert '*) ;; *) die 'The selected file is not a Localert binary.' ;; esac
stop_existing
mv -f "$temporary" "$target"

cat > "$launcher" <<EOF
#!/bin/sh
umask 077
printf '%s\n' "\$\$" > $(quote_sh "$pid_file")
crash=$(quote_sh "$log_dir/crash.log")
# Only crash traces reach stderr; keep that file small too.
if [ -f "\$crash" ] && [ "\$(wc -c < "\$crash")" -gt 1048576 ]; then mv -f "\$crash" "\$crash.1"; fi
exec $(quote_sh "$target") --data-dir $(quote_sh "$data_dir") --pid-file $(quote_sh "$pid_file") --log-file $(quote_sh "$log_file") >/dev/null 2>> "\$crash"
EOF
chmod 700 "$launcher"

if [ "$os" = linux ]; then
  cat > "$autostart" <<EOF
[Desktop Entry]
Type=Application
Name=Localert
Comment=Local desktop notification API
Exec=/bin/sh "$(desktop_escape "$launcher")"
Terminal=false
X-GNOME-Autostart-enabled=true
StartupNotify=false
EOF
  chmod 600 "$autostart"
  if [ "$start" -eq 1 ]; then
    nohup /bin/sh "$launcher" </dev/null >/dev/null 2>&1 &
  fi
else
  cat > "$autostart" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
  <key>Label</key><string>dev.localert.agent</string>
  <key>ProgramArguments</key><array><string>/bin/sh</string><string>$(xml_escape "$launcher")</string></array>
  <key>RunAtLoad</key><true/>
  <key>KeepAlive</key><true/>
  <key>ThrottleInterval</key><integer>10</integer>
  <key>LimitLoadToSessionType</key><string>Aqua</string>
  <key>ProcessType</key><string>Interactive</string>
  <key>StandardOutPath</key><string>/dev/null</string>
  <key>StandardErrorPath</key><string>/dev/null</string>
</dict></plist>
EOF
  chmod 600 "$autostart"
  if [ "$start" -eq 1 ]; then
    command -v launchctl >/dev/null 2>&1 || die 'launchctl is unavailable; sign in to a macOS desktop session to start Localert.'
    launchctl bootstrap "gui/$uid" "$autostart"
  fi
fi

say "Installed $binary_version"
say "Binary:    $target"
say "Database:  $data_dir/localert.db"
say "Autostart: $autostart"
say "Log:       $log_file"
say 'Dashboard: http://127.0.0.1:8787'
if [ "$start" -eq 1 ]; then say 'Localert is starting. The browser opens automatically if password setup is needed.'; else say 'Localert will start the next time you sign in.'; fi
