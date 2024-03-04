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
    # install missing packages
    need_install = @packages.reject { |p| package? p }
    need_install_args = need_install.join(" ")
    if need_install.any?
      log "Installing packages", packages: need_install
      sudo "pacman --noconfirm --needed -S #{need_install_args}"
    end

    # expand groups to packages
    packages_args = @packages.join(" ")
    group_packages = Set.new(`pacman --quiet -Sg #{packages_args}`.lines.map(&:strip))

    # full list of packages that should exist on the system
    all = @packages + group_packages

    # actual list on the system
    installed = Set.new(`pacman -Q --quiet --explicit --unrequired --native`.lines.map(&:strip))

    unneeded = installed - all
    next if unneeded.empty?

    log "Removing packages", packages: unneeded
    sudo("pacman -Rs #{unneeded.join(" ")}")
  end
end

# aur command to install packages from aur on install step
def aur(*names)
  names.flatten!
  @aurs ||= Set.new
  @aurs += names.map(&:to_s)

  on_install do
    names = @aurs || []
    log "Install AUR packages", packages: names
    cache = "./cache/aur"
    FileUtils.mkdir_p cache
    Dir.chdir cache do
      names.each do |package|
        system("git clone --depth 1 --shallow-submodules https://aur.archlinux.org/#{package}.git") unless Dir.exist?(package)
        Dir.chdir package do

          pkgbuild = File.readlines('PKGBUILD')
          pkgver = pkgbuild.find { |l| l.start_with?('pkgver=') }.split('=')[1].strip.chomp('"')
          package_info = `pacman -Qi #{package}`.strip.lines.map{|l| l.strip.split(/\s*:\s*/, 2) }.to_h
          installed = package_info["Version"].to_s.split("-")[0] == pkgver

          system("makepkg --syncdeps --install --noconfirm --needed") unless installed
        end
      end
    end
  end
end
