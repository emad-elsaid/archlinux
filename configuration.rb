linux do
  timezone 'Europe/Berlin'
  keyboard keymap: 'us',
           layout: "us,ara",
           model: "",
           variant: "",
           options: "ctrl:nocaps,caps:lctrl,ctrl:swap_lalt_lctl,grp:alt_space_toggle"

  # Core ==========
  package 'linux',
          'linux-firmware',
          'linux-headers',
          'base',
          'base-devel',
          'bash-completion',
          'pacman-contrib'

  # Networking =====
  package 'dnsutils',
          'dhcpcd',
          'dhcp',
          'network-manager-applet',
          'networkmanager',
          'networkmanager-openvpn',
          'nm-connection-editor',
          'pulseaudio',
          'pulseaudio-alsa',
          'alsa-utils',
          'pulseaudio-bluetooth',
          'traceroute',
          'ufw',
          'avahi',
          'iftop',
          'nmap'

  # Hardware ========
  package 'acpi',
          'blueman',
          'bluez',
          'bluez-utils',
          'fwupd',
          'guvcview',
          'pavucontrol',
          'usbutils',
          'v4l2loopback-dkms',
          'exfat-utils',
          'intel-ucode',
          'system-config-printer',
          'xf86-video-intel'

  # Display =======
  package 'xorg',
          'xorg-apps',
          'xorg-fonts',
          'xorg-xinit',
          'arandr',
          'autorandr',
          'i3',
          'picom'

  # Services =====
  package 'mate-notification-daemon',
          'syncthing',
          'dbus'

  # CLIs =========
  package 'sudo',
          'time',
          'bat',
          'cloc',
          'calc',
          'ctop',
          'iotop',
          'htop',
          'dialog',
          'fzf',
          'grep',
          'gzip',
          'jq',
          'lshw',
          'lsof',
          'ncdu',
          'pv',
          'ranger',
          'rsync',
          'scrot',
          'the_silver_searcher',
          'tree',
          'unrar',
          'unzip',
          'wget',
          'whois',
          'xclip',
          'xdotool',
          'zip',
          'locate',
          'jpegoptim',
          'nano',
          'powertop'

  # Apps =========
  package 'cheese',
          'eog',
          'eog-plugins',
          'gnome-disk-utility',
          'gnome-keyring',
          'nautilus',
          'konsole',
          'feh',
          'gimp',
          'inkscape',
          'hexchat',
          'mplayer',
          'obs-studio',
          'vlc',
          'keepassxc',
          'keybase-gui',
          'rawtherapee',
          'shotwell',
          'totem'

  # Themes ========
  package 'lxappearance-gtk3',
          'elementary-icon-theme',
          'gtk-theme-elementary',
          'arc-gtk-theme',
          'capitaine-cursors',
          'inter-font',
          'noto-fonts',
          'noto-fonts-emoji',
          'powerline-fonts',
          'ttf-dejavu',
          'ttf-font-awesome',
          'ttf-jetbrains-mono',
          'ttf-liberation',
          'ttf-ubuntu-font-family'

  # Dev tools =====
  package 'android-tools',
          'make',
          'autoconf',
          'automake',
          'clang',
          'dbeaver',
          'jdk-openjdk',
          'emacs',
          'neovim',
          'docker',
          'docker-compose',
          'github-cli',
          'graphviz',
          'git'

  # Prog. langs ===
  package 'ruby',
          'go',
          'python'

  # Utils =========
  package 'aspell-en',
          'man-pages'

  # Libs =========
  package 'postgresql-libs'

  aur 'kernel-install-mkinitcpio',
      'polybar-git',
      'xbanish',
      'xlog-git',
      'units',
      'google-chrome',
      'ttf-amiri',
      'ttf-mac-fonts',
      'ttf-sil-lateef',
      'ttf-arabeyes-fonts',
      'ttf-meslo',
      'aspell-ar',
      'autoenv-git',
      'siji-git'

  service 'avahi-daemon',
          'bluetooth',
          'docker',
          'NetworkManager',
          'systemd-boot-update',
          'systemd-fsck-root',
          'systemd-homed',
          'systemd-network-generator',
          'systemd-networkd-wait-online',
          'systemd-networkd',
          'systemd-pstore',
          'systemd-remount-fs',
          'systemd-resolved',
          'systemd-timesyncd',
          'ufw'

  user_service 'ssh-agent',
               'syncthing',
               'keybase',
               'kbfs'

  timer 'plocate-updatedb',
        'checkupdates'

  group 'wheel', sudo: true
  user 'emad', groups: ['wheel', 'docker']
  run 'sudo ufw enable syncthing'

  run 'sudo bootctl install'
  run 'sudo reinstall-kernels'

  sync './root/etc', '/'
  sync '/user/.config', '/home/emad/.config'
  replace_line '/etc/mkinitcpio.conf', /^(.*)base udev(.*)$/, "$1systemd$2"

end
