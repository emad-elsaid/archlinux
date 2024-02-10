require_relative 'lib'

linux do
  timedate timezone: 'Europe/Berlin',
           ntp: true

  keyboard keymap: 'us',
           layout: "us,ara",
           model: "",
           variant: "",
           options: "ctrl:nocaps,caps:lctrl,ctrl:swap_lalt_lctl,grp:alt_space_toggle"

  # Core ==========
  package %w[
          linux
          linux-firmware
          linux-headers
          base
          base-devel
          bash-completion
          pacman-contrib
          ]

  # Networking =====
  package %w[
          dhcpcd
          dhcp
          network-manager-applet
          networkmanager
          networkmanager-openvpn
          nm-connection-editor
          pulseaudio
          pulseaudio-alsa
          alsa-utils
          pulseaudio-bluetooth
          traceroute
          avahi
          iftop
          nmap
          ]

  # Hardware ========
  package %w[
          acpi
          blueman
          bluez
          bluez-utils
          fwupd
          guvcview
          pavucontrol
          usbutils
          v4l2loopback-dkms
          exfat-utils
          intel-ucode
          system-config-printer
          xf86-video-intel
          ]

  # Display =======
  package %w[
          xorg
          xorg-apps
          xorg-fonts
          xorg-xinit
          arandr
          autorandr
          i3
          picom
          ]

  # Services =====
  package %w[
          mate-notification-daemon
          syncthing
          dbus
          ]

  # CLIs =========
  package %w[
          sudo
          time
          bat
          cloc
          calc
          ctop
          iotop
          htop
          dialog
          fzf
          grep
          gzip
          jq
          lshw
          lsof
          ncdu
          pv
          ranger
          rsync
          scrot
          the_silver_searcher
          tree
          unrar
          unzip
          wget
          whois
          xclip
          xdotool
          zip
          locate
          jpegoptim
          nano
          powertop
          ]

  # Apps =========
  package %w[
          cheese
          eog
          eog-plugins
          gnome-disk-utility
          gnome-keyring
          nautilus
          konsole
          feh
          gimp
          inkscape
          hexchat
          mplayer
          obs-studio
          vlc
          keepassxc
          keybase-gui
          rawtherapee
          shotwell
          totem
          ]

  # Themes ========
  package %w[
          lxappearance-gtk3
          elementary-icon-theme
          gtk-theme-elementary
          arc-gtk-theme
          capitaine-cursors
          inter-font
          noto-fonts
          noto-fonts-emoji
          powerline-fonts
          ttf-dejavu
          ttf-font-awesome
          ttf-jetbrains-mono
          ttf-liberation
          ttf-ubuntu-font-family
          ]

  # Dev tools =====
  package %w[
          android-tools
          make
          autoconf
          automake
          clang
          dbeaver
          jdk-openjdk
          emacs
          neovim
          docker
          docker-compose
          github-cli
          graphviz
          git
          ]

  # Prog. langs ===
  package %w[ruby go python]

  # Utils =========
  package %w[aspell-en man-pages]

  # Libs =========
  package 'postgresql-libs'

  service %w[
          avahi-daemon
          bluetooth
          docker
          NetworkManager
          ]

  timer 'plocate-updatedb'

  user 'emad', groups: ['wheel', 'docker'] do
    aur %w[
        kernel-install-mkinitcpio
        polybar-git
        xbanish
        xlog-git
        units
        google-chrome
        ttf-amiri
        ttf-mac-fonts
        ttf-sil-lateef
        ttf-arabeyes-fonts
        ttf-meslo
        aspell-ar
        autoenv-git
        siji-git
        ]

    service %w[
            ssh-agent
            syncthing
            keybase
            kbfs
            ]

    copy './user/.config/.', '/home/emad/.config'
    timer 'checkupdates'
  end

  firewall :syncthing

  on_finalize do
    sudo 'bootctl install'
    sudo 'reinstall-kernels'
  end

  copy './root/etc', '/'
  replace '/etc/mkinitcpio.conf', /^(.*)base udev(.*)$/, '\1systemd\2'
end
