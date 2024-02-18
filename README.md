# Archlinux

[![Gem Version](https://badge.fury.io/rb/archlinux.svg)](https://badge.fury.io/rb/archlinux)

> [!WARNING]
> this can break your system, don't use it on your running system

A ruby API to manage the state of an Archlinux system.

* This project is early alpha
* It aims to have a DSL like NixOS for Archlinux
* It allow declaring the state of the system then it applies that to the current system
* The idea is to have simple and user friendly API to declare everything, system and user related

## Getting started

The project is a Ruby Gem. Create a ruby file and require the gem:

```ruby
require 'bundler/inline'

gemfile do
  source "https://rubygems.org"

  gem "archlinux", github: "emad-elsaid/archlinux"
end
```

Use `linux` function to define your system state, for example:

```ruby
linux do
  hostname 'earth'

  timedate timezone: 'Europe/Berlin',
           ntp: true

  locale "en_US.UTF-8"
  keyboard keymap: 'us',
           layout: "us,ara",
           model: "",
           variant: "",
           options: "ctrl:nocaps,caps:lctrl,ctrl:swap_lalt_lctl,grp:alt_space_toggle"

  package %w[
          linux
          linux-firmware
          linux-headers
          base
          base-devel
          bash-completion
          pacman-contrib
          docker
          locate
          syncthing
          ]

  service %w[
          docker
          NetworkManager
          ]

  timer 'plocate-updatedb'

  user 'smith', groups: ['wheel', 'docker'] do
    aur %w[
        kernel-install-mkinitcpio
        google-chrome
        ]

    service %w[
            ssh-agent
            syncthing
            ]

    copy './user/.config/.', '/home/smith/.config'
  end

  firewall :syncthing

  on_finalize do
    sudo 'bootctl install'
    sudo 'reinstall-kernels'
  end

  file '/etc/X11/xorg.conf.d/40-touchpad.conf', <<~EOT
  Section "InputClass"
          Identifier "libinput touchpad catchall"
          MatchIsTouchpad "on"
          MatchDevicePath "/dev/input/event*"
          Driver "libinput"
          Option "Tapping" "on"
          Option "NaturalScrolling" "true"
  EndSection
  EOT

  replace '/etc/mkinitcpio.conf', /^(.*)base udev(.*)$/, '\1systemd\2'
end
```

Now you can run the script with ruby as root:

```shell
sudo ruby <script-name.rb>
```


It will do the following:
- Install missing packages, remove any other package
- Make sure services and timers are running
- Do other configurations like locale, X11 keyboard settings, hostname
- Ensure users are created and in specified groups
