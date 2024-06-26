require 'set'
require 'fileutils'

# @group Utilities

# Utility function, returns true of package is installed
def package?(name)
  system("pacman -Qi #{name} &> /dev/null")
end

# @group Declarations:

# Install a package on install step and remove packages not registered with this
# function
def package(*names)
  names.flatten!
  @packages ||= Set.new
  @packages += names.map(&:to_s)

  # install step to install packages required and remove not required
  on_install do
    # Expand groups to packages
    packages_args = @packages.join(" ")
    group_packages = Set.new(`pacman --quiet -Sg #{packages_args}`.lines.map(&:strip))

    all = @packages + group_packages # full list of packages that should exist on the system

    # actual list on the system
    installed = Set.new(`pacman -Q --quiet --explicit --unrequired --native`.lines.map(&:strip))

    unneeded = installed - all
    if unneeded.any?
      log "Removing packages", packages: unneeded
      sudo("pacman -Rsu #{unneeded.join(" ")}")
    end

    # install missing packages
    need_install = @packages.reject { |p| package? p }
    need_install_args = need_install.join(" ")
    if need_install.any?
      log "Installing packages", packages: need_install
      sudo "pacman --noconfirm --needed -S #{need_install_args}"
    end
  end
end

# aur command to install packages from aur on install step
def aur(*names)
  names.flatten!
  @aurs ||= Set.new
  @aurs += names.map(&:to_s)

  on_install do
    log "Install AUR packages", packages: @aurs
    cache = "./cache/aur"
    FileUtils.mkdir_p cache
    Dir.chdir cache do
      @aurs.each do |package|
        unless Dir.exist?(package)
          system("git clone --depth 1 --shallow-submodules https://aur.archlinux.org/#{package}.git")
        end
        Dir.chdir package do
          pkgbuild = File.readlines('PKGBUILD')
          pkgver = pkgbuild.find { |l| l.start_with?('pkgver=') }.split('=')[1].strip.chomp('"')
          package_info = `pacman -Qi #{package}`.strip.lines.to_h { |l| l.strip.split(/\s*:\s*/, 2) }
          installed = package_info["Version"].to_s.split("-")[0] == pkgver

          system("makepkg --syncdeps --install --noconfirm --needed") unless installed
        end
      end
    end

    foreign = Set.new(`pacman -Qm`.lines.map { |l| l.split(/\s+/, 2).first })
    unneeded = foreign - @aurs
    next if unneeded.empty?

    log "Foreign packages to remove", packages: unneeded
    sudo("pacman -Rsu #{unneeded.join(" ")}")
  end
end
