require 'set'
require 'fileutils'

# @group Declarations:
# Functions the user will run to declare the state of the system
# like packages to be present, files, services, user, group...etc

# Install a package on install step and remove packages not registered with this
# function
def package(*names)
  names.flatten!
  @packages ||= Set.new
  @packages += names.map(&:to_s)

  # install step to install packages required and remove not required
  on_install do
    # install packages list as is
    names = @packages.join(" ")
    log "Installing packages", packages: @packages
    sudo "pacman --noconfirm --needed -S #{names}" unless @packages.empty?

    # expand groups to packages
    group_packages = Set.new(`pacman --quiet -Sg #{names}`.lines.map(&:strip))

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
        system("git clone --depth 1 --shallow-submodules https://aur.archlinux.org/#{package}.git") unless Dir.exists?(package)
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

# set timezone and NTP settings during prepare step
def timedate(timezone: 'UTC', ntp: true)
  @timedate = {timezone: timezone, ntp: ntp}

  on_configure do
    log "Set timedate", @timedate
    sudo "timedatectl set-timezone #{@timedate[:timezone]}"
    sudo "timedatectl set-ntp #{@timedate[:ntp]}"
  end
end

# enable system service if root or user service if not during finalize step
def service(*names)
  names.flatten!
  @services ||= Set.new
  @services += names.map(&:to_s)

  on_finalize do
    log "Enable services", services: @services
    user_flags = root? ? "" : "--user"

    system "systemctl enable #{user_flags} #{@services.join(" ")}"

    # Disable services that were enabled manually and not in the list we have
    services = `systemctl list-unit-files #{user_flags} --state=enabled --type=service --no-legend --no-pager`
    enabled_manually = services.lines.map{|l| l.strip.split(/\s+/) }.select{|l| (l[1] == 'enabled') && (l[2] == 'disabled')}
    names_without_extension = enabled_manually.map{|l| l.first.delete_suffix(".service") }
    to_disable = names_without_extension - @services.to_a

    next if to_disable.empty?

    log "Services to disable", packages: to_disable
    # system "systemctl disable #{user_flags} #{to_disable.join(" ")}"
  end
end

# enable system timer if root or user timer if not during finalize step
def timer(*names)
  names.flatten!
  @timers ||= Set.new
  @timers += names.map(&:to_s)

  on_finalize do
    log "Enable timers", timers: @timers
    timers = @timers.map{ |t| "#{t}.timer" }.join(" ")
    if root?
      sudo "systemctl enable #{timers}"
    else
      system "systemctl enable --user #{timers}"
    end
    # disable all other timers
  end
end

# set keyboard settings during prepare step
def keyboard(keymap: nil, layout: nil, model: nil, variant: nil, options: nil)
  @keyboard ||= {}
  values = {
    keymap: keymap,
    layout: layout,
    model: model,
    variant: variant,
    options: options
  }.compact
  @keyboard.merge!(values)

  on_prepare do
    next unless @keyboard[:keymap]

    sudo "localectl set-keymap #{@keyboard[:keymap]}"

    m = @keyboard.to_h.slice(:layout, :model, :variant, :options)
    sudo "localectl set-x11-keymap \"#{m[:layout]}\" \"#{m[:model]}\" \"#{m[:variant]}\" \"#{m[:options]}\""
  end
end

# Sets locale using localectl
def locale(value)
  @locale = value

  on_prepare do
    sudo "localectl set-locale #{@locale}"
  end
end

# create a user and assign a set of group. if block is passes the block will run
# in as this user. block will run during the configure step
def user(name, groups: [], &block)
  name = name.to_s

  @user ||= {}
  @user[name] ||= {}
  @user[name][:groups] ||= []
  @user[name][:groups] += groups.map(&:to_s)
  @user[name][:state] = State.new
  @user[name][:state].apply(block) if block_given?

  on_configure do
    @user.each do |name, conf|
      exists = Etc.getpwnam name rescue nil
      sudo "useradd #{name}" unless exists
      sudo "usermod --groups #{groups.join(",")} #{name}" if groups.any?

      fork do
        currentuser = Etc.getpwnam(name)
        Process::GID.change_privilege(currentuser.gid)
        Process::UID.change_privilege(currentuser.uid)
        ENV['XDG_RUNTIME_DIR'] = "/run/user/#{currentuser.uid}"
        ENV['HOME'] = currentuser.dir
        ENV['USER'] = currentuser.name
        ENV['LOGNAME'] = currentuser.name
        conf[:state].run_steps
      end

      Process.wait
    end
  end
end

# Copy src inside dest during configure step, if src/. will copy src content to dest
def copy(src, dest)
  @copy ||= []
  @copy << { src: src, dest: dest }

  on_configure do
    next unless @copy
    next if @copy.empty?

    @copy.each do |item|
      log "Copying", item
      FileUtils.cp_r item[:src], item[:dest]
    end
  end
end

# Replace a regex pattern with replacement string in a file during configure step
def replace(file, pattern, replacement)
  @replace ||= []
  @replace << {file: file, pattern: pattern, replacement: replacement}

  on_configure do
    @replace.each do |params|
      input = File.read(params[:file])
      output = input.gsub(params[:pattern], params[:replacement])
      File.write(params[:file], output)
    end
  end
end

# setup add ufw enable it and allow ports during configure step
def firewall(*allow)
  @firewall ||= Set.new
  @firewall += allow.map(&:to_s)

  package :ufw
  service :ufw

  on_configure do
    sudo "ufw allow #{@firewall.join(' ')}"
  end
end

# Write a file during configure step
def file(path, content)
  @files ||= {}
  @files[path] = content

  on_configure do
    @files.each do |path, content|
      File.write(path, content)
    end
  end
end


# Sets the machine hostname
def hostname(name)
  @hostname = name

  file '/etc/hostname', "#{@hostname}\n"

  on_configure do
    log "Setting hostname", hostname: @hostname
    sudo "hostnamectl set-hostname #{@hostname}"
  end
end

# link file to destination
def symlink(target, link_name)
  @symlink ||= Set.new
  @symlink << {target: target, link_name: link_name}

  on_configure do

    @symlink.each do |params|
      target = File.expand_path params[:target]
      link_name = File.expand_path params[:link_name]
      log "Linking", target: target, link_name: link_name

      # make the parent if it doesn't exist
      dest_dir = File.dirname(link_name)
      FileUtils.mkdir_p(dest_dir) unless File.exist?(dest_dir)

      # link with force
      FileUtils.ln_s(target, link_name, force: true)
    end
  end
end

# on prepare make sure the directory exists
def mkdir(*path)
  path.flatten!
  @mkdir ||= Set.new
  @mkdir += path

  on_prepare do
    @mkdir.each do |path|
      FileUtils.mkdir_p File.expand_path(path)
    end
  end
end

# on prepare make sure a git repository is cloned to directory
def git_clone(from:, to: nil)
  @git_clone ||= Set.new
  @git_clone << {from: from, to: to}

  on_install do
    @git_clone.each do |item|
      from = item[:from]
      to = item[:to]
      system "git clone #{from} #{to}" unless File.exists?(File.expand_path(to))
    end
  end
end

# git clone for github repositories
def github_clone(from:, to: nil)
  git_clone(from: "https://github.com/#{from}", to: to)
end
